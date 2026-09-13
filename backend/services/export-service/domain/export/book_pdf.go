package export

const bookPDFMediaType = "application/pdf"

// BookPDFProjection carries rendered bytes and diagnostics. Browser lifecycle
// and protocol code belong to the service's infrastructure adapter.
type BookPDFProjection struct {
	Content      []byte
	MediaType    string
	RenderLog    string
	ToolVersions map[string]string
	FontEvidence *PDFFontEvidence
}
