package httpapp

// RemiStaticAssetVersion is appended as ?v=... to shared static assets (app.css, app.js, vendored HTMX)
// in HTML templates. Bump when users need a fresh fetch without hard refresh.
const RemiStaticAssetVersion = "311"

// RemiStaticAssetV is a template.FuncMap helper for href="/static/app.css?v={{remiStaticAssetV}}".
func RemiStaticAssetV() string {
	return RemiStaticAssetVersion
}
