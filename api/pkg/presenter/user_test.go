package presenter

import (
	"context"
	"errors"
	"testing"

	"github.com/mmhk/svn-admin/pkg/model"
)

func TestUserList(t *testing.T) {
	presenter := newUserPresenter()

	list, err := presenter.List(context.Background(), model.PageQuery{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if list.Page != 1 || list.Size != model.DefaultPageSize {
		t.Errorf("page/size = %d/%d, want 1/%d", list.Page, list.Size, model.DefaultPageSize)
	}

	if list.Total != 2 || len(list.Items) != 2 {
		t.Fatalf("total/items = %d/%d, want 2/2", list.Total, len(list.Items))
	}

	if list.Items[0].Name != "admin" || list.Items[1].Name != "alice" {
		t.Errorf("order = %q, %q; want admin, alice", list.Items[0].Name, list.Items[1].Name)
	}

	alice := list.Items[1]
	if len(alice.Groups) != 1 || alice.Groups[0] != "devs" {
		t.Errorf("alice groups = %v", alice.Groups)
	}
}

func TestUserListFilterAndPaging(t *testing.T) {
	presenter := newUserPresenter()

	filtered, err := presenter.List(context.Background(), model.PageQuery{Q: "ali"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if filtered.Total != 1 || len(filtered.Items) != 1 || filtered.Items[0].Name != "alice" {
		t.Errorf("filtered = %+v", filtered)
	}

	paged, err := presenter.List(context.Background(), model.PageQuery{Page: 2, Size: 1})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if paged.Total != 2 || len(paged.Items) != 1 || paged.Items[0].Name != "alice" {
		t.Errorf("paged = %+v", paged)
	}

	clamped, err := presenter.List(context.Background(), model.PageQuery{Size: 100000})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if clamped.Size != model.MaxPageSize {
		t.Errorf("Size = %d, want clamped to %d", clamped.Size, model.MaxPageSize)
	}
}

func TestUserGetNotFound(t *testing.T) {
	presenter := newUserPresenter()

	_, err := presenter.Get(context.Background(), "nobody")
	if err == nil {
		t.Fatal("Get() error = nil, want not found")
	}

	var notFound *model.NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("error = %T %v, want *model.NotFoundError", err, err)
	}
}

func TestUserGetValidatesName(t *testing.T) {
	presenter := newUserPresenter()

	_, err := presenter.Get(context.Background(), "../etc/passwd")
	if err == nil {
		t.Fatal("Get() error = nil, want validation error")
	}

	var validation *model.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %T %v, want *model.ValidationError", err, err)
	}
}
