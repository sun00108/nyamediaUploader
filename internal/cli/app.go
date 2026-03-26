package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"nyamediaUploader/internal/auth"
	"nyamediaUploader/internal/mediarequest"
	"nyamediaUploader/internal/onedrive"
	"nyamediaUploader/internal/ui"
	"nyamediaUploader/internal/uploader"
)

type App struct {
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader
	auth   *auth.Service
	od     *onedrive.Client
	req    *mediarequest.Client
	store  *mediarequest.Store
}

func New() *App {
	cfg := auth.LoadConfig()

	return &App{
		stdout: os.Stdout,
		stderr: os.Stderr,
		stdin:  os.Stdin,
		auth:   auth.NewService(cfg),
		od:     onedrive.NewClient(cfg.BotAPIBaseURL),
		req:    mediarequest.NewClient(cfg.BotAPIBaseURL),
		store:  mediarequest.NewStore(cfg.ConfigDir),
	}
}

func (a *App) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		a.printHelp()
		return nil
	}

	switch args[0] {
	case "login":
		return a.runLogin(ctx)
	case "logout":
		return a.runLogout(ctx)
	case "request":
		return a.runRequest(ctx, args[1:])
	case "upload":
		return a.runUpload(ctx, args[1:])
	case "help", "-h", "--help":
		a.printHelp()
		return nil
	default:
		a.printHelp()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func (a *App) runLogin(ctx context.Context) error {
	session, err := a.auth.LoadSession(ctx)
	if err != nil && !errors.Is(err, auth.ErrSessionNotFound) {
		return err
	}

	if session != nil && session.IsValidAt(auth.Now()) {
		fmt.Fprintf(a.stdout, "Already logged in as %s\n", session.DisplayUser())
		fmt.Fprintf(a.stdout, "Token file: %s\n", a.auth.SessionPath())
		return nil
	}

	loginURL, state, err := a.auth.BeginLogin(ctx)
	if err != nil {
		return err
	}

	fmt.Fprintln(a.stdout, "Open this URL in your browser and complete Telegram login:")
	fmt.Fprintln(a.stdout, loginURL)
	fmt.Fprintln(a.stdout)
	fmt.Fprintln(a.stdout, "After login, paste the authorization code here.")

	code, err := auth.ReadAuthorizationCode(a.stdin, a.stdout)
	if err != nil {
		return err
	}

	session, err = a.auth.CompleteLogin(ctx, auth.CompleteLoginInput{
		State:             state,
		AuthorizationCode: code,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(a.stdout, "Login successful. Signed in as %s\n", session.DisplayUser())
	fmt.Fprintf(a.stdout, "Token file: %s\n", a.auth.SessionPath())
	return nil
}

func (a *App) runLogout(ctx context.Context) error {
	session, err := a.auth.LoadSession(ctx)
	if err != nil && !errors.Is(err, auth.ErrSessionNotFound) {
		return err
	}

	if session == nil {
		fmt.Fprintln(a.stdout, "Already logged out.")
		return nil
	}

	if err := a.auth.Logout(ctx); err != nil {
		return err
	}

	fmt.Fprintln(a.stdout, "Logged out.")
	return nil
}

func (a *App) runRequest(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("request", flag.ContinueOnError)
	fs.SetOutput(a.stderr)

	var requestID int64
	var season int
	var episode int

	fs.Int64Var(&requestID, "request-id", 0, "media_requests.id from bot")
	fs.IntVar(&season, "season", 0, "season number for TV requests")
	fs.IntVar(&episode, "episode", 0, "episode number for TV requests")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if requestID <= 0 {
		return errors.New("missing or invalid required flag: --request-id")
	}

	session, err := a.requireSession(ctx)
	if err != nil {
		return err
	}

	var seasonPtr *int
	if season > 0 {
		seasonPtr = &season
	}
	var episodePtr *int
	if episode > 0 {
		episodePtr = &episode
	}

	resp, err := a.req.Create(ctx, session.AccessToken, mediarequest.CreateInput{
		RequestID: requestID,
		Season:    seasonPtr,
		Episode:   episodePtr,
	})
	if err != nil {
		return err
	}

	item := mediarequest.Item{
		RequestID:   requestID,
		RequestCode: resp.RequestCode,
		MediaTitle:  resp.MediaTitle,
		Season:      resp.Season,
		Episode:     resp.Episode,
		CreatedAt:   time.Now(),
	}
	if err := a.store.Add(ctx, item); err != nil {
		return err
	}

	fmt.Fprintln(a.stdout, "Upload request created.")
	fmt.Fprintf(a.stdout, "Title: %s\n", formatRequestLabel(item))
	fmt.Fprintf(a.stdout, "Request Code: %s\n", item.RequestCode)
	fmt.Fprintf(a.stdout, "Store: %s\n", a.store.Path())
	return nil
}

func (a *App) runUpload(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("upload", flag.ContinueOnError)
	fs.SetOutput(a.stderr)

	var conflictBehavior string
	fs.StringVar(&conflictBehavior, "conflict-behavior", "replace", "replace, rename, or fail")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: nyaupload upload [file]")
	}
	localFile := fs.Arg(0)

	info, err := os.Stat(localFile)
	if err != nil {
		return fmt.Errorf("stat local file %s: %w", localFile, err)
	}
	if info.IsDir() {
		return fmt.Errorf("local file %s is a directory", localFile)
	}
	if info.Size() <= 0 {
		return fmt.Errorf("local file %s is empty", localFile)
	}

	session, err := a.requireSession(ctx)
	if err != nil {
		return err
	}

	items, err := a.store.Load(ctx)
	if err != nil {
		if errors.Is(err, mediarequest.ErrRequestStoreNotFound) {
			return errors.New("no local upload requests found, run `nyaupload request` first")
		}
		return err
	}
	if len(items) == 0 {
		return errors.New("no local upload requests found, run `nyaupload request` first")
	}

	options := make([]string, 0, len(items))
	for _, item := range items {
		options = append(options, formatRequestLabel(item))
	}
	index, err := ui.Select("Select an upload request", options, a.stdout)
	if err != nil {
		return err
	}
	selected := items[index]
	uploadFileName := buildUploadFileName(selected, localFile)

	resp, err := a.od.CreateUploadSession(ctx, session.AccessToken, onedrive.CreateUploadSessionInput{
		RequestCode:      selected.RequestCode,
		FileName:         uploadFileName,
		FileSize:         info.Size(),
		ConflictBehavior: conflictBehavior,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(a.stdout, "Uploading %s\n", localFile)
	if resp.Path != "" {
		fmt.Fprintf(a.stdout, "Remote Path: %s\n", resp.Path)
	}
	result, err := uploader.UploadFile(ctx, localFile, resp.UploadURL, a.stdout)
	if err != nil {
		return err
	}

	if _, err := a.req.CompleteUpload(ctx, session.AccessToken, mediarequest.CompleteUploadInput{
		RequestCode: selected.RequestCode,
		FileName:    uploadFileName,
	}); err != nil {
		return err
	}

	if err := a.store.RemoveByCode(ctx, selected.RequestCode); err != nil {
		return err
	}

	fmt.Fprintln(a.stdout, "Upload completed.")
	fmt.Fprintf(a.stdout, "Status Code: %d\n", result.StatusCode)
	fmt.Fprintf(a.stdout, "Request: %s\n", formatRequestLabel(selected))
	return nil
}

func (a *App) requireSession(ctx context.Context) (*auth.Session, error) {
	session, err := a.auth.LoadSession(ctx)
	if err != nil {
		if errors.Is(err, auth.ErrSessionNotFound) {
			return nil, errors.New("not logged in, run `nyaupload login` first")
		}
		return nil, err
	}
	if !session.IsValidAt(auth.Now()) {
		return nil, errors.New("local session is expired, run `nyaupload login` again")
	}
	return session, nil
}

func formatRequestLabel(item mediarequest.Item) string {
	parts := []string{item.MediaTitle}
	if item.Season != nil {
		parts = append(parts, "S"+pad2(*item.Season))
	}
	if item.Episode != nil {
		parts = append(parts, "E"+pad2(*item.Episode))
	}
	parts = append(parts, "request_id="+strconv.FormatInt(item.RequestID, 10))
	return strings.Join(parts, " ")
}

func pad2(v int) string {
	if v < 10 {
		return "0" + strconv.Itoa(v)
	}
	return strconv.Itoa(v)
}

func buildUploadFileName(item mediarequest.Item, localFile string) string {
	ext := filepath.Ext(localFile)
	if item.Season != nil && item.Episode != nil {
		return fmt.Sprintf("%s - S%sE%s%s", item.MediaTitle, pad2(*item.Season), pad2(*item.Episode), ext)
	}
	return item.MediaTitle + ext
}

func (a *App) printHelp() {
	fmt.Fprintln(a.stdout, "nyaupload CLI")
	fmt.Fprintln(a.stdout)
	fmt.Fprintln(a.stdout, "Usage:")
	fmt.Fprintln(a.stdout, "  nyaupload login")
	fmt.Fprintln(a.stdout, "  nyaupload logout")
	fmt.Fprintln(a.stdout, "  nyaupload request --request-id 123 [--season 1 --episode 2]")
	fmt.Fprintln(a.stdout, "  nyaupload upload ./movie.mkv")
	fmt.Fprintln(a.stdout)
	fmt.Fprintln(a.stdout, "Environment overrides:")
	fmt.Fprintln(a.stdout, "  NYAUPLOAD_BOT_PUBLIC_BASE_URL")
	fmt.Fprintln(a.stdout, "  NYAUPLOAD_BOT_API_BASE_URL")
	fmt.Fprintln(a.stdout, "  NYAUPLOAD_CONFIG_DIR")
	fmt.Fprintln(a.stdout, "  NYAUPLOAD_CLIENT_ID")
}
