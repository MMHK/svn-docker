package presenter

import (
	"context"
	"strings"

	"github.com/mmhk/svn-admin/pkg/model"
	"github.com/mmhk/svn-admin/pkg/view"
)

// AccessPath serves authz access-path read and write use cases.
type AccessPath struct {
	paths  model.AccessPathStore
	writer model.AccessPathWriter
}

// NewAccessPath builds the access-path presenter.
func NewAccessPath(
	paths model.AccessPathStore,
	writer model.AccessPathWriter,
) *AccessPath {
	return &AccessPath{paths: paths, writer: writer}
}

// List returns a page of access paths with ACLs.
func (p *AccessPath) List(ctx context.Context, query model.PageQuery) (view.AccessPathList, error) {
	page, err := p.paths.ListAccessPaths(ctx, query.Normalize())
	if err != nil {
		return view.AccessPathList{}, err
	}

	items := make([]view.AccessPath, 0, len(page.Items))
	for _, path := range page.Items {
		items = append(items, accessPathView(path))
	}

	return view.AccessPathList{
		Items: items,
		ListMeta: view.ListMeta{
			Page:  page.Page,
			Size:  page.Size,
			Total: page.Total,
		},
	}, nil
}

// Get returns one access path and its ACL.
func (p *AccessPath) Get(ctx context.Context, path string) (view.AccessPath, error) {
	if err := (model.AccessPath{Path: path}).Validate(); err != nil {
		return view.AccessPath{}, err
	}

	accessPath, err := p.paths.GetAccessPath(ctx, path)
	if err != nil {
		return view.AccessPath{}, err
	}

	return accessPathView(accessPath), nil
}

// Create adds an empty access-path section.
func (p *AccessPath) Create(ctx context.Context, path string, opts model.WriteOptions) (view.AccessPath, error) {
	path = strings.TrimSpace(path)

	if err := (model.AccessPath{Path: path}).Validate(); err != nil {
		return view.AccessPath{}, err
	}

	if err := p.writer.CreateAccessPath(ctx, path, opts); err != nil {
		return view.AccessPath{}, err
	}

	return view.AccessPath{Path: path, ACL: []view.ACLEntry{}}, nil
}

// Delete removes the section and every ACL entry in it.
func (p *AccessPath) Delete(ctx context.Context, path string, opts model.WriteOptions) error {
	path = strings.TrimSpace(path)

	if err := (model.AccessPath{Path: path}).Validate(); err != nil {
		return err
	}

	return p.writer.DeleteAccessPath(ctx, path, opts)
}

// SetACL creates or replaces one principal/permission pair.
func (p *AccessPath) SetACL(
	ctx context.Context,
	path, principal string,
	permission model.Permission,
	opts model.WriteOptions,
) (view.AccessPath, error) {
	path = strings.TrimSpace(path)
	principal = strings.TrimSpace(principal)

	if err := p.writer.SetACL(ctx, path, principal, permission, opts); err != nil {
		return view.AccessPath{}, err
	}

	return p.Get(ctx, path)
}

// DeleteACL removes one principal/permission pair.
func (p *AccessPath) DeleteACL(ctx context.Context, path, principal string, opts model.WriteOptions) error {
	path = strings.TrimSpace(path)
	principal = strings.TrimSpace(principal)

	return p.writer.DeleteACL(ctx, path, principal, opts)
}

func accessPathView(path model.AccessPath) view.AccessPath {
	acl := make([]view.ACLEntry, 0, len(path.ACL))
	for _, entry := range path.ACL {
		acl = append(acl, view.ACLEntry{Principal: entry.Principal, Permission: string(entry.Permission)})
	}

	return view.AccessPath{Path: path.Path, ACL: acl}
}
