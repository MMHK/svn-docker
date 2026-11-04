package httpapi

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/mmhk/svn-admin/pkg/auth"
	"github.com/mmhk/svn-admin/pkg/model"
	"github.com/mmhk/svn-admin/pkg/view"
)

// P4 write-action names for the stderr operation log (plan T2.5).
const (
	actionRepositoryCreate = "repository.create"
	actionRepositoryDelete = "repository.delete"
)

// createRepository godoc
//
//	@Summary		Create a repository
//	@Description	Runs `svnadmin create --fs-type fsfs` in the first writable SVNParentPath, optionally commits the standard layout ("single" = trunk/branches/tags, "multi" = <projectName>/trunk|branches|tags) and optionally creates the `<name>:/` authz section. The owner of the new repository is the service uid; nothing is chowned.
//	@Tags			repositories
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			request	body		view.CreateRepositoryRequest	true	"repository name and optional layout"
//	@Success		201		{object}	view.RepositoryCreated
//	@Failure		400		{object}	view.ErrorEnvelope
//	@Failure		401		{object}	view.ErrorEnvelope
//	@Failure		403		{object}	view.ErrorEnvelope	"role does not permit repository:add"
//	@Failure		409		{object}	view.ErrorEnvelope	"a repository with this name already exists"
//	@Failure		422		{object}	view.ErrorEnvelope	"invalid name, structure or projectName"
//	@Failure		500		{object}	view.ErrorEnvelope	"svnadmin failed; the failed step is reported"
//	@Failure		501		{object}	view.ErrorEnvelope	"the Subversion CLI is not configured in this deployment"
//	@Failure		503		{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/repos [post]
func (a *API) createRepository(w http.ResponseWriter, r *http.Request) {
	if err := a.authorize(r, auth.ModuleRepository, auth.ActionAdd); err != nil {
		a.writeError(w, r, err)

		return
	}

	var request view.CreateRepositoryRequest
	if err := decodeJSON(w, r, &request); err != nil {
		a.auditWrite(r, actionRepositoryCreate, request.Name, err)
		a.writeError(w, r, err)

		return
	}

	created, err := a.repo.Create(r.Context(), request)
	if err != nil {
		a.auditWrite(r, actionRepositoryCreate, request.Name, err)
		a.writeError(w, r, err)

		return
	}

	a.auditWrite(r, actionRepositoryCreate, created.Name, nil)

	w.Header().Set("Location", "/svnadmin/api/v1/repos/"+url.PathEscape(created.Name)+"/info")
	writeJSON(w, http.StatusCreated, created)
}

// deleteRepository godoc
//
//	@Summary		Delete a repository
//	@Description	Moves the repository into SVN_ADMIN_TRASH_DIR (default `<data>/trash`) as `<repo>-<UTC20060102T150405Z>`; never removes data recursively. The caller must repeat the repository name in `confirm`. Cross-filesystem moves are refused with 409 and need a manual move. Deletion is enabled by default and disabled with SVN_ADMIN_REPO_DELETE_ENABLED=false.
//	@Tags			repositories
//	@Security		bearerAuth
//	@Param			repo	path	string	true	"repository name"
//	@Param			confirm	query	string	true	"must equal the repository name"
//	@Success		204
//	@Failure		400		{object}	view.ErrorEnvelope
//	@Failure		401		{object}	view.ErrorEnvelope
//	@Failure		403		{object}	view.ErrorEnvelope	"role does not permit repository:delete, or deletion is disabled"
//	@Failure		404		{object}	view.ErrorEnvelope
//	@Failure		409		{object}	view.ErrorEnvelope	"the repository is on another filesystem than the trash directory"
//	@Failure		422		{object}	view.ErrorEnvelope	"confirm does not match the repository name"
//	@Failure		503		{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/repos/{repo} [delete]
func (a *API) deleteRepository(w http.ResponseWriter, r *http.Request) {
	if err := a.authorize(r, auth.ModuleRepository, auth.ActionDelete); err != nil {
		a.writeError(w, r, err)

		return
	}

	name := r.PathValue("repo")
	confirm := r.URL.Query().Get("confirm")

	if err := a.repo.Delete(r.Context(), name, confirm); err != nil {
		a.auditWrite(r, actionRepositoryDelete, name, err)
		a.writeError(w, r, err)

		return
	}

	a.auditWrite(r, actionRepositoryDelete, name, nil)

	w.WriteHeader(http.StatusNoContent)
}

// getRepositoryTree godoc
//
//	@Summary		Browse a repository directory
//	@Description	Runs `svn list --xml` and returns one page of entries. `path` must be relative and must not contain '.' or '..' segments; `rev` is a positive revision number, empty means HEAD. Entries are sorted by name before paging (default 100, max 1000). Subversion stderr and filesystem paths are never returned.
//	@Tags			repositories
//	@Produce		json
//	@Security		bearerAuth
//	@Param			repo	path		string	true	"repository name"
//	@Param			path	query		string	false	"relative directory path; empty means the repository root"
//	@Param			rev		query		int		false	"revision; empty means HEAD"
//	@Param			page	query		int		false	"page number (default 1)"
//	@Param			size	query		int		false	"page size (default 100, max 1000)"
//	@Success		200		{object}	view.RepositoryTree
//	@Failure		401		{object}	view.ErrorEnvelope
//	@Failure		403		{object}	view.ErrorEnvelope
//	@Failure		404		{object}	view.ErrorEnvelope	"repository or path not found"
//	@Failure		422		{object}	view.ErrorEnvelope	"invalid path, revision or paging"
//	@Failure		500		{object}	view.ErrorEnvelope
//	@Failure		501		{object}	view.ErrorEnvelope	"the Subversion CLI is not configured in this deployment"
//	@Failure		503		{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/repos/{repo}/tree [get]
func (a *API) getRepositoryTree(w http.ResponseWriter, r *http.Request) {
	if err := a.authorize(r, auth.ModuleRepository, auth.ActionView); err != nil {
		a.writeError(w, r, err)

		return
	}

	revision, err := parseRevision(r.URL.Query().Get("rev"))
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	query, err := parseTreePageQuery(r)
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	tree, err := a.repo.Tree(
		r.Context(),
		r.PathValue("repo"),
		r.URL.Query().Get("path"),
		revision,
		query,
	)
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	writeJSON(w, http.StatusOK, tree)
}

// getRepositoryInfo godoc
//
//	@Summary		Repository information
//	@Description	HEAD revision plus the last-changed revision, author and date. HEAD comes from `svn log -r HEAD --limit 1`; `svn info` supplies the last change, because its entry revision is not the repository HEAD.
//	@Tags			repositories
//	@Produce		json
//	@Security		bearerAuth
//	@Param			repo	path		string	true	"repository name"
//	@Success		200		{object}	view.Repository
//	@Failure		401		{object}	view.ErrorEnvelope
//	@Failure		403		{object}	view.ErrorEnvelope
//	@Failure		404		{object}	view.ErrorEnvelope
//	@Failure		422		{object}	view.ErrorEnvelope
//	@Failure		500		{object}	view.ErrorEnvelope
//	@Failure		501		{object}	view.ErrorEnvelope	"the Subversion CLI is not configured in this deployment"
//	@Failure		503		{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/repos/{repo}/info [get]
func (a *API) getRepositoryInfo(w http.ResponseWriter, r *http.Request) {
	if err := a.authorize(r, auth.ModuleRepository, auth.ActionView); err != nil {
		a.writeError(w, r, err)

		return
	}

	info, err := a.repo.Info(r.Context(), r.PathValue("repo"))
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	writeJSON(w, http.StatusOK, info)
}

// parseRevision reads the optional rev query. Empty means HEAD (0).
func parseRevision(raw string) (int64, error) {
	if raw == "" {
		return 0, nil
	}

	revision, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || revision < 1 {
		return 0, &model.ValidationError{Field: "rev", Reason: "must be a positive integer"}
	}

	return revision, nil
}

// parseTreePageQuery reads page/size for the browser. It deliberately does not
// reuse parsePageQuery because the tree defaults are 100/1000 rather than
// 50/500.
func parseTreePageQuery(r *http.Request) (model.PageQuery, error) {
	query := r.URL.Query()
	page := model.PageQuery{}

	if raw := query.Get("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return model.PageQuery{}, &model.ValidationError{Field: "page", Reason: "must be an integer"}
		}

		page.Page = parsed
	}

	if raw := query.Get("size"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return model.PageQuery{}, &model.ValidationError{Field: "size", Reason: "must be an integer"}
		}

		page.Size = parsed
	}

	return page, nil
}
