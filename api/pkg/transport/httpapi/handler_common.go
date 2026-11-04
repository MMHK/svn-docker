package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/mmhk/svn-admin/pkg/model"
)

const maxRequestBody = 1 << 20

// decodeJSON binds the request body, rejecting unknown fields so typos cannot
// be silently ignored. The Content-Type is not enforced because curl -d sends
// application/x-www-form-urlencoded while still writing a JSON body.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return &model.ValidationError{Field: "body", Reason: "must be a valid JSON object"}
	}

	return nil
}

// parsePageQuery reads q/page/size. Non-numeric values are a validation error;
// out-of-range values are clamped by model.PageQuery.Normalize.
func parsePageQuery(r *http.Request) (model.PageQuery, error) {
	query := r.URL.Query()
	page := model.PageQuery{Q: query.Get("q")}

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

	return page.Normalize(), nil
}

// accessPathFromRequest accepts either ?path= or the URL-safe base64 path
// segment from plan section 4.2.
func accessPathFromRequest(r *http.Request) (string, error) {
	if value := r.URL.Query().Get("path"); value != "" {
		return value, nil
	}

	raw := r.PathValue("path")
	if raw == "" {
		return "", &model.ValidationError{Field: "path", Reason: "is required"}
	}

	for _, encoding := range []*base64.Encoding{
		base64.RawURLEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.StdEncoding,
	} {
		if decoded, err := encoding.DecodeString(raw); err == nil {
			return string(decoded), nil
		}
	}

	return "", &model.ValidationError{Field: "path", Reason: "must be URL-safe base64"}
}
