package scripts

const scriptsTable = "scripts"

const (
	scriptsTableColumnID         = "id"
	scriptsTableColumnFilename   = "filename"
	scriptsTableColumnPath       = "path"
	scriptsTableColumnUploadTime = "upload_time"
	scriptsTableColumnText       = "text"
	scriptsTableColumnUserID     = "user_id"
)

var scriptsTableColumns = []string{
	scriptsTableColumnID,
	scriptsTableColumnFilename,
	scriptsTableColumnPath,
	scriptsTableColumnUploadTime,
	scriptsTableColumnText,
	scriptsTableColumnUserID,
}
