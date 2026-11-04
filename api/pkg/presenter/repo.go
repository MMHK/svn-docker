package presenter

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/mmhk/svn-admin/pkg/model"
	"github.com/mmhk/svn-admin/pkg/view"
)

// Tree paging defaults from the rewrite plan (section 4.1 / T4.2): the
// directory browser pages at 100 entries and never returns more than 1000.
const (
	treeDefaultPageSize = 100
	treeMaxPageSize     = 1000
)

// RepositoryDeps are the collaborators of the repository use cases. Writer and
// Browser are optional so the P1 read-only wiring keeps working; a missing one
// turns the corresponding write into model.ErrReadOnly instead of a panic.
type RepositoryDeps struct {
	Repos       model.RepositoryStore
	Writer      model.RepositoryWriter
	Browser     model.RepositoryBrowser
	AccessPaths model.AccessPathWriter
	// DeleteEnabled mirrors SVN_ADMIN_REPO_DELETE_ENABLED (plan D20: default
	// true, false returns 403).
	DeleteEnabled bool
}

// Repository serves repository read and write use cases.
type Repository struct {
	repos       model.RepositoryStore
	writer      model.RepositoryWriter
	browser     model.RepositoryBrowser
	accessPaths model.AccessPathWriter
	deleteOK    bool
}

// NewRepository builds the repository presenter.
func NewRepository(deps RepositoryDeps) *Repository {
	return &Repository{
		repos:       deps.Repos,
		writer:      deps.Writer,
		browser:     deps.Browser,
		accessPaths: deps.AccessPaths,
		deleteOK:    deps.DeleteEnabled,
	}
}

// List returns a page of repositories.
func (p *Repository) List(ctx context.Context, query model.PageQuery) (view.RepositoryList, error) {
	page, err := p.repos.ListRepositories(ctx, query.Normalize())
	if err != nil {
		return view.RepositoryList{}, err
	}

	items := make([]view.Repository, 0, len(page.Items))
	for _, repository := range page.Items {
		items = append(items, repositoryView(repository))
	}

	return view.RepositoryList{
		Items: items,
		ListMeta: view.ListMeta{
			Page:  page.Page,
			Size:  page.Size,
			Total: page.Total,
		},
	}, nil
}

// Get returns one repository.
func (p *Repository) Get(ctx context.Context, name string) (view.Repository, error) {
	if err := (model.Repository{Name: name}).Validate(); err != nil {
		return view.Repository{}, err
	}

	repository, err := p.repos.GetRepository(ctx, name)
	if err != nil {
		return view.Repository{}, err
	}

	return repositoryView(repository), nil
}

// Create runs `svnadmin create`, the optional layout commit and the optional
// `repo:/` access-path section. Once the repository exists, a later failure is
// reported as an OperationError naming the steps that already ran, so a caller
// never mistakes a half-built repository for success (plan T4.3, T4.4).
func (p *Repository) Create(ctx context.Context, request view.CreateRepositoryRequest) (view.RepositoryCreated, error) {
	structure, err := model.ParseRepositoryStructure(request.Structure)
	if err != nil {
		return view.RepositoryCreated{}, err
	}

	spec := model.RepositorySpec{
		Name:        strings.TrimSpace(request.Name),
		Structure:   structure,
		ProjectName: strings.TrimSpace(request.ProjectName),
	}

	if err := spec.Validate(); err != nil {
		return view.RepositoryCreated{}, err
	}

	if p.writer == nil {
		return view.RepositoryCreated{}, model.ErrReadOnly
	}

	repository, err := p.writer.CreateRepository(ctx, spec)
	if err != nil {
		return view.RepositoryCreated{}, err
	}

	completed := []string{"svnadmin-create"}
	if len(structure.LayoutDirs(spec.ProjectName)) > 0 {
		completed = append(completed, "layout")
	}

	if request.AccessPath {
		accessPath := spec.Name + ":/"

		if p.accessPaths == nil {
			return view.RepositoryCreated{}, &model.OperationError{
				Operation: "repository.create",
				Step:      "access-path",
				Completed: completed,
				Err:       model.ErrReadOnly,
			}
		}

		if err := p.accessPaths.CreateAccessPath(ctx, accessPath, model.WriteOptions{}); err != nil {
			return view.RepositoryCreated{}, &model.OperationError{
				Operation: "repository.create",
				Step:      "access-path",
				Completed: completed,
				Err:       err,
			}
		}

		completed = append(completed, "access-path")
	}

	return view.RepositoryCreated{
		Name:             repository.Name,
		Parent:           repository.Parent,
		URL:              repository.URL,
		CreatedStructure: string(structure),
	}, nil
}

// Delete moves a repository into the trash directory after the caller repeated
// its name (plan T4.5 / D7 / D20).
func (p *Repository) Delete(ctx context.Context, name, confirm string) error {
	name = strings.TrimSpace(name)

	if err := (model.Repository{Name: name}).Validate(); err != nil {
		return err
	}

	if confirm != name {
		return &model.ValidationError{
			Field:  "confirm",
			Reason: "must repeat the repository name exactly",
		}
	}

	if !p.deleteOK {
		return &model.ForbiddenError{Reason: "repository deletion is disabled by configuration"}
	}

	if p.writer == nil {
		return model.ErrReadOnly
	}

	if _, err := p.writer.DeleteRepository(ctx, name); err != nil {
		return err
	}

	return nil
}

// Tree returns one page of a repository directory. Entries are sorted by name
// and then sliced, so the page is stable regardless of `svn list` order
// (plan T4.2).
func (p *Repository) Tree(
	ctx context.Context,
	name, path string,
	revision int64,
	query model.PageQuery,
) (view.RepositoryTree, error) {
	name = strings.TrimSpace(name)
	path = strings.TrimSpace(path)

	if err := (model.Repository{Name: name}).Validate(); err != nil {
		return view.RepositoryTree{}, err
	}

	if err := model.ValidateRepositoryPath(path); err != nil {
		return view.RepositoryTree{}, err
	}

	if revision < 0 {
		return view.RepositoryTree{}, &model.ValidationError{
			Field:  "rev",
			Reason: "must be a positive integer",
		}
	}

	if p.browser == nil {
		return view.RepositoryTree{}, model.ErrReadOnly
	}

	tree, err := p.browser.Tree(ctx, name, path, revision)
	if err != nil {
		return view.RepositoryTree{}, err
	}

	entries := make([]model.RepositoryEntry, len(tree.Entries))
	copy(entries, tree.Entries)

	sort.SliceStable(entries, func(i, j int) bool {
		left, right := strings.ToLower(entries[i].Name), strings.ToLower(entries[j].Name)
		if left == right {
			return entries[i].Name < entries[j].Name
		}

		return left < right
	})

	page, size := normalizeTreePage(query)
	total := len(entries)
	start := (page - 1) * size
	if start > total {
		start = total
	}

	end := start + size
	if end > total {
		end = total
	}

	items := make([]view.RepositoryEntry, 0, end-start)
	for _, entry := range entries[start:end] {
		items = append(items, view.RepositoryEntry{
			Name:     entry.Name,
			Kind:     entry.Kind,
			Size:     entry.Size,
			Revision: entry.Revision,
			Author:   entry.Author,
			Date:     optionalTime(entry.Date),
		})
	}

	return view.RepositoryTree{
		Path:     tree.Path,
		Revision: tree.Revision,
		Entries:  items,
		ListMeta: view.ListMeta{Page: page, Size: size, Total: total},
	}, nil
}

// Info returns the HEAD revision plus the last-changed metadata (plan T4.6).
func (p *Repository) Info(ctx context.Context, name string) (view.Repository, error) {
	name = strings.TrimSpace(name)

	if err := (model.Repository{Name: name}).Validate(); err != nil {
		return view.Repository{}, err
	}

	if p.browser == nil {
		return view.Repository{}, model.ErrReadOnly
	}

	repository, err := p.browser.Info(ctx, name)
	if err != nil {
		return view.Repository{}, err
	}

	return repositoryView(repository), nil
}

// normalizeTreePage applies the browser defaults, which differ from the list
// defaults (100/1000 instead of 50/500).
func normalizeTreePage(query model.PageQuery) (page, size int) {
	page, size = query.Page, query.Size

	if page < 1 {
		page = 1
	}

	if size <= 0 {
		size = treeDefaultPageSize
	}

	if size > treeMaxPageSize {
		size = treeMaxPageSize
	}

	return page, size
}

// optionalTime keeps zero timestamps out of the JSON payload.
func optionalTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}

	utc := value.UTC()

	return &utc
}

func repositoryView(repository model.Repository) view.Repository {
	return view.Repository{
		Name:              repository.Name,
		Parent:            repository.Parent,
		URL:               repository.URL,
		IsRepository:      repository.IsRepository,
		HeadRevision:      repository.HeadRevision,
		LastChangedRev:    repository.LastChangedRev,
		LastChangedAuthor: repository.LastChangedAuthor,
		LastChangedAt:     timePtr(repository.LastChangedAt),
	}
}
