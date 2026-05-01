package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"text/tabwriter"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/AlecAivazis/survey/v2"
	"github.com/google/go-github/v60/github"
	"github.com/sirupsen/logrus"
	gitlab "github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
	"github.com/joho/godotenv"
)

const (
	defaultStorageDir   = "./storage"
	defaultSyncInterval = 1 * time.Hour
	defaultWorkerCount  = 5
)

type SyncStatus string

const (
	StatusNoRemote  SyncStatus = "NO_REMOTE"
	StatusOutOfSync SyncStatus = "OUT_OF_SYNC"
	StatusSynced    SyncStatus = "SYNCED"
)

type RepoStatus struct {
	Repo   *github.Repository
	Status SyncStatus
}

func main() {
	listMode := flag.Bool("list", false, "Apenas mostra a tabela comparativa e encerra")
	interactiveMode := flag.Bool("interactive", false, "Abre o menu de seleção interativo")
	allMode := flag.Bool("all", false, "Roda o fluxo automático para todos os repositórios")
	flag.Parse()

	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		logrus.Debug("No .env file found")
	}

	// Initialize Logrus
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		ForceColors:     true,
		TimestampFormat: "15:04:05",
	})
	logrus.SetLevel(logrus.InfoLevel)

	githubToken := os.Getenv("GITHUB_TOKEN")
	gitlabToken := os.Getenv("GITLAB_TOKEN")
	giteaToken := os.Getenv("GITEA_TOKEN")

	if githubToken == "" {
		logrus.Fatal("Error: GITHUB_TOKEN environment variable not set")
	}

	storageDir := os.Getenv("STORAGE_DIR")
	if storageDir == "" {
		storageDir = defaultStorageDir
	}

	workerCount := defaultWorkerCount
	if v := os.Getenv("WORKER_COUNT"); v != "" {
		fmt.Sscanf(v, "%d", &workerCount)
	}

	if err := os.MkdirAll(storageDir, 0755); err != nil {
		logrus.Fatalf("Error creating storage directory: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	githubClient := initGitHubClient(githubToken)

	var gitlabClient *gitlab.Client
	if gitlabToken != "" {
		var err error
		if gitlabURL := os.Getenv("GITLAB_BASE_URL"); gitlabURL != "" && gitlabURL != "https://gitlab.com" {
			gitlabClient, err = gitlab.NewClient(gitlabToken, gitlab.WithBaseURL(gitlabURL+"/api/v4"))
		} else {
			gitlabClient, err = gitlab.NewClient(gitlabToken)
		}
		if err != nil {
			logrus.Fatalf("Error creating GitLab client: %v", err)
		}
	}

	var giteaClient *gitea.Client
	if giteaToken != "" {
		giteaURL := os.Getenv("GITEA_URL")
		if giteaURL == "" {
			giteaURL = "http://localhost:3000"
		}
		var err error
		giteaClient, err = gitea.NewClient(giteaURL, gitea.SetToken(giteaToken))
		if err != nil {
			logrus.Fatalf("Error creating Gitea client: %v", err)
		}
	}

	// Fetch GitHub and GitLab info
	githubRepos, err := fetchGitHubRepos(ctx, githubClient)
	if err != nil {
		logrus.Errorf("Error fetching GitHub repos: %v", err)
	}
	if githubRepos == nil {
		logrus.Fatal("Não foi possível carregar repositórios do GitHub. Encerrando.")
	}

	gitlabProjects, err := fetchGitLabRepos(ctx, gitlabClient)
	if err != nil {
		logrus.Errorf("Error fetching GitLab projects (continuing without GitLab info): %v", err)
	}

	repoStatuses := compareRepos(githubRepos, gitlabProjects)

	if *listMode {
		printStatusTable(repoStatuses)
		return
	}

	var reposToSync []RepoStatus

	if *interactiveMode {
		reposToSync = selectRepos(repoStatuses)
		if len(reposToSync) == 0 {
			logrus.Info("Nenhum repositório selecionado. Encerrando.")
			return
		}
		var confirm bool
		survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("%d repositórios selecionados. Prosseguir?", len(reposToSync)),
			Default: true,
		}, &confirm)
		if !confirm {
			return
		}
	} else if *allMode {
		reposToSync = repoStatuses
	} else {
		logrus.Info("Nenhum modo selecionado. Use --list, --interactive, ou --all para processar repositórios.")
		return
	}

	if err := syncReposList(ctx, reposToSync, githubClient, gitlabClient, giteaClient, storageDir, workerCount, *interactiveMode); err != nil {
		logrus.Errorf("Erro durante a sincronização: %v", err)
	} else {
		logrus.Info("Sincronização concluída com sucesso.")
	}
}

func initGitHubClient(token string) *github.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	return github.NewClient(tc)
}

func fetchGitHubRepos(ctx context.Context, client *github.Client) ([]*github.Repository, error) {
	logrus.Info("Fetching GitHub repositories...")
	var allRepos []*github.Repository
	opt := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := client.Repositories.List(ctx, "", opt)
		if err != nil {
			return nil, err
		}
		// Rate limit check
		if resp.Rate.Remaining < 100 {
			logrus.Warnf("Restam %d chamadas no Rate Limit do GitHub", resp.Rate.Remaining)
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			logrus.Infof("Restam %d chamadas no Rate Limit do GitHub", resp.Rate.Remaining)
			break
		}
		opt.Page = resp.NextPage
	}
	logrus.Infof("Found %d repositories on GitHub", len(allRepos))
	return allRepos, nil
}

func fetchGitLabRepos(ctx context.Context, client *gitlab.Client) (map[string]bool, error) {
	if client == nil {
		return nil, nil
	}
	logrus.Info("Fetching GitLab projects...")
	gitlabProjects := make(map[string]bool)
	opt := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
		Owned:       github.Bool(true),
	}
	for {
		projects, resp, err := client.Projects.ListProjects(opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, err
		}
		for _, p := range projects {
			gitlabProjects[p.Path] = true
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	logrus.Infof("Found %d projects on GitLab", len(gitlabProjects))
	return gitlabProjects, nil
}

func compareRepos(githubRepos []*github.Repository, gitlabProjects map[string]bool) []RepoStatus {
	var statuses []RepoStatus
	for _, repo := range githubRepos {
		status := StatusNoRemote
		// Note: p.Path in GitLab is equivalent to repoName in GitHub (usually). 
		// Actually, gitlabProjects stores p.Path.
		if gitlabProjects != nil && gitlabProjects[*repo.Name] {
			status = StatusOutOfSync // Assume it needs sync if it exists
		}
		statuses = append(statuses, RepoStatus{
			Repo:   repo,
			Status: status,
		})
	}
	return statuses
}

func printStatusTable(statuses []RepoStatus) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "REPOSITORY\tSTATUS\tPRIVATE")
	fmt.Fprintln(w, "----------\t------\t-------")
	for _, rs := range statuses {
		priv := "No"
		if *rs.Repo.Private {
			priv = "Yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", *rs.Repo.Name, rs.Status, priv)
	}
	w.Flush()
}

func selectRepos(allRepos []RepoStatus) []RepoStatus {
	var options []string
	repoMap := make(map[string]RepoStatus)
	for _, rs := range allRepos {
		label := fmt.Sprintf("%s [%s]", *rs.Repo.Name, rs.Status)
		options = append(options, label)
		repoMap[label] = rs
	}

	var selectedLabels []string
	prompt := &survey.MultiSelect{
		Message: "Quais repositórios você deseja sincronizar?",
		Options: options,
	}
	survey.AskOne(prompt, &selectedLabels)

	var selectedRepos []RepoStatus
	for _, label := range selectedLabels {
		selectedRepos = append(selectedRepos, repoMap[label])
	}
	return selectedRepos
}

func syncReposList(ctx context.Context, repos []RepoStatus, githubClient *github.Client, gitlabClient *gitlab.Client, giteaClient *gitea.Client, storageDir string, workerCount int, interactive bool) error {
	repoChan := make(chan RepoStatus, len(repos))
	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for rs := range repoChan {
				select {
				case <-ctx.Done():
					return
				default:
					if err := processRepository(ctx, rs.Repo, githubClient, gitlabClient, giteaClient, storageDir, interactive); err != nil {
						logrus.Errorf("[Worker %d] Error processing repository %s: %v", workerID, *rs.Repo.Name, err)
					}
				}
			}
		}(i)
	}

	for _, rs := range repos {
		repoChan <- rs
	}
	close(repoChan)

	wg.Wait()
	return nil
}

func processRepository(ctx context.Context, repo *github.Repository, githubClient *github.Client, gitlabClient *gitlab.Client, giteaClient *gitea.Client, storageDir string, interactive bool) error {
	repoName := *repo.Name
	repoURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s.git", os.Getenv("GITHUB_TOKEN"), *repo.FullName)
	isPrivate := *repo.Private

	startTime := time.Now()
	logrus.Infof("Processing repository: %s (private: %v)", repoName, isPrivate)

	var mirrorURLs []string

	if gitlabClient != nil {
		gitlabURL := fmt.Sprintf("https://oauth2:%s@gitlab.com/%s/%s.git", os.Getenv("GITLAB_TOKEN"), getGitHubLogin(), repoName)
		mirrorURLs = append(mirrorURLs, gitlabURL)

		if err := ensureGitLabRepoExists(ctx, gitlabClient, repoName, isPrivate, interactive); err != nil {
			logrus.Warnf("Skipping GitLab sync for %s: %v", repoName, err)
			mirrorURLs = mirrorURLs[:len(mirrorURLs)-1]
		}
	}

	if giteaClient != nil {
		giteaSSHHost := os.Getenv("GITEA_SSH_HOST")
		if giteaSSHHost == "" {
			giteaSSHHost = "localhost"
		}
		giteaSSHURL := fmt.Sprintf("git@%s:%s/%s.git", giteaSSHHost, getGitHubLogin(), repoName)
		mirrorURLs = append(mirrorURLs, giteaSSHURL)

		if err := ensureGiteaRepoExists(ctx, giteaClient, repoName, isPrivate, interactive); err != nil {
			logrus.Warnf("Skipping Gitea sync for %s: %v", repoName, err)
			mirrorURLs = mirrorURLs[:len(mirrorURLs)-1]
		}
	}

	if len(mirrorURLs) > 0 {
		if err := syncRepo(ctx, repoName, repoURL, mirrorURLs, storageDir); err != nil {
			return fmt.Errorf("error syncing repository %s: %w", repoName, err)
		}
	}

	elapsed := time.Since(startTime)
	logrus.Infof("Repo %s sincronizado em %.2fs", repoName, elapsed.Seconds())

	return nil
}

func syncRepo(ctx context.Context, repoName string, githubURL string, mirrorURLs []string, storageDir string) error {
	localPath := filepath.Join(storageDir, fmt.Sprintf("%s.git", repoName))

	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		logrus.Infof("[%s] Cloning mirror...", repoName)
		cmd := exec.CommandContext(ctx, "git", "clone", "--mirror", githubURL, localPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to clone repository %s: %w (output: %s)", repoName, err, string(output))
		} else if len(output) > 0 {
			logrus.Infof("[%s] Clone concluído:\n%s", repoName, string(output))
		}
	} else {
		logrus.Infof("[%s] Updating local mirror...", repoName)
		cmd := exec.CommandContext(ctx, "git", "-C", localPath, "remote", "update")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to update repository %s: %w (output: %s)", repoName, err, string(output))
		} else if len(output) > 0 {
			logrus.Infof("[%s] Update concluído:\n%s", repoName, string(output))
		}
	}

	for _, mirrorURL := range mirrorURLs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			logrus.Infof("[%s] Pushing to %s...", repoName, maskURL(mirrorURL))
			cmd := exec.CommandContext(ctx, "git", "-C", localPath, "push", "--mirror", mirrorURL)
			if output, err := cmd.CombinedOutput(); err != nil {
				logrus.Warnf("[%s] Failed to push to %s: %v (output: %s)", repoName, maskURL(mirrorURL), err, string(output))
			} else {
				logrus.Infof("[%s] Push concluído para %s:\n%s", repoName, maskURL(mirrorURL), string(output))
			}
		}
	}

	return nil
}

func ensureGitLabRepoExists(ctx context.Context, client *gitlab.Client, repoName string, isPrivate bool, interactive bool) error {
	_, _, err := client.Projects.GetProject(getGitHubLogin()+"/"+repoName, nil)
	if err == nil {
		return nil
	}

	if interactive {
		var confirm bool
		errSurvey := survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Repo '%s' não existe no GitLab. Criar?", repoName),
			Default: true,
		}, &confirm)
		if errSurvey != nil {
			return errSurvey
		}
		if !confirm {
			return fmt.Errorf("criação cancelada pelo usuário")
		}
	}

	logrus.Infof("Creating GitLab repository: %s (private: %v)", repoName, isPrivate)
	visibility := gitlab.PrivateVisibility
	if !isPrivate {
		visibility = gitlab.PublicVisibility
	}

	repoOptions := &gitlab.CreateProjectOptions{
		Name:                 github.String(repoName),
		Visibility:           &visibility,
		InitializeWithReadme: github.Bool(false),
	}

	_, _, err = client.Projects.CreateProject(repoOptions, gitlab.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to create GitLab repository %s: %w", repoName, err)
	}

	return nil
}

func ensureGiteaRepoExists(ctx context.Context, client *gitea.Client, repoName string, isPrivate bool, interactive bool) error {
	user, _, err := client.GetUserInfo("")
	if err != nil {
		return fmt.Errorf("failed to get Gitea user info: %w", err)
	}

	_, _, err = client.GetRepo(user.UserName, repoName)
	if err == nil {
		return nil
	}

	if interactive {
		var confirm bool
		errSurvey := survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Repo '%s' não existe no Gitea. Criar?", repoName),
			Default: true,
		}, &confirm)
		if errSurvey != nil {
			return errSurvey
		}
		if !confirm {
			return fmt.Errorf("criação cancelada pelo usuário")
		}
	}

	logrus.Infof("Creating Gitea repository: %s/%s (private: %v)", user.UserName, repoName, isPrivate)
	createOpt := gitea.CreateRepoOption{
		Name:     repoName,
		Private:  isPrivate,
		AutoInit: false,
	}

	_, _, err = client.CreateRepo(createOpt)
	if err != nil {
		return fmt.Errorf("failed to create Gitea repository %s/%s: %w", user.UserName, repoName, err)
	}

	return nil
}

func getGitHubLogin() string {
	if login := os.Getenv("GITHUB_LOGIN"); login != "" {
		return login
	}
	return "Apenasgabs"
}

func maskURL(u string) string {
	// Procura o padrão :token@ e substitui o token por ***
	// Ex: https://oauth2:MEU_TOKEN@gitlab.com -> https://oauth2:***@gitlab.com
	start := -1
	for i := 0; i < len(u); i++ {
		if u[i] == ':' {
			start = i
		}
		if u[i] == '@' && start != -1 {
			return u[:start+1] + "***" + u[i:]
		}
	}
	return u
}
