package export

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func (r *GormRepository) loadRightsLedger(ctx context.Context, buildID uuid.UUID, manifestHash string, originals []sharedtextbook.RightsItem) (sharedtextbook.RightsLedger, error) {
	var rows []struct {
		ID                  uuid.UUID
		BuildID             uuid.UUID
		BaseItemID          uuid.UUID
		PreviousAmendmentID *uuid.UUID
		Revision            int
		ManifestHash        string
		DocumentJSON        datatypes.JSON
		CompletedAt         time.Time
	}
	if err := r.db.WithContext(ctx).Table("textbook_rights_amendments").Where("build_id = ?", buildID).Order("base_item_id ASC, revision ASC").Find(&rows).Error; err != nil {
		return sharedtextbook.RightsLedger{}, err
	}
	history := make([]sharedtextbook.RightsAmendment, 0, len(rows))
	for _, row := range rows {
		var a sharedtextbook.RightsAmendment
		previous := ""
		if row.PreviousAmendmentID != nil {
			previous = row.PreviousAmendmentID.String()
		}
		if json.Unmarshal(row.DocumentJSON, &a) != nil || a.ID != row.ID.String() || a.BuildID != row.BuildID.String() || a.BaseItemID != row.BaseItemID.String() || a.PreviousAmendmentID != previous || a.Revision != row.Revision || a.ManifestHash != row.ManifestHash || !a.CompletedAt.Equal(row.CompletedAt) {
			return sharedtextbook.RightsLedger{}, ErrTextbookBookBuildInvalid
		}
		history = append(history, a)
	}
	ledger, err := sharedtextbook.ResolveRightsLedger(buildID.String(), manifestHash, originals, history)
	if err != nil {
		return sharedtextbook.RightsLedger{}, ErrTextbookBookBuildInvalid
	}
	return ledger, nil
}
