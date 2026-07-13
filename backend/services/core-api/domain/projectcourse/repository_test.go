package projectcourse

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newCourseTestRepository(t *testing.T) *GormRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:project-course-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&ProjectCourse{}); err != nil {
		t.Fatal(err)
	}
	return NewGormRepository(db)
}

func TestRepositoryEnforcesOwnerIsolationAndBlueprintCAS(t *testing.T) {
	repo := newCourseTestRepository(t)
	owner, other := uuid.New(), uuid.New()
	course := &ProjectCourse{UserID: owner, RepositoryURL: "https://github.com/example/project", RequestedRef: "main", ResolvedCommitSHA: "0123456789abcdef0123456789abcdef01234567", AudienceLevel: "programming", Status: StatusAwaitingApproval, BlueprintVersion: 1, BlueprintJSON: datatypes.JSON(`{}`), CoverageJSON: datatypes.JSON(`{}`), QualityReportJSON: datatypes.JSON(`{}`)}
	if err := repo.Create(context.Background(), course); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetByID(context.Background(), other, course.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-owner read: %v", err)
	}
	if err := repo.UpdateBlueprintCAS(context.Background(), owner, course.ID, BlueprintUpdate{ExpectedVersion: 0, BlueprintJSON: []byte(`{"x":1}`), CoverageJSON: []byte(`{}`)}); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected version conflict, got %v", err)
	}
	if err := repo.UpdateBlueprintCAS(context.Background(), owner, course.ID, BlueprintUpdate{ExpectedVersion: 1, BlueprintJSON: []byte(`{"x":1}`), CoverageJSON: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Approve(context.Background(), owner, course.ID, 2); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateBlueprintCAS(context.Background(), owner, course.ID, BlueprintUpdate{ExpectedVersion: 2, BlueprintJSON: []byte(`{"x":2}`), CoverageJSON: []byte(`{}`)}); !errors.Is(err, ErrBlueprintImmutable) {
		t.Fatalf("approved blueprint should be immutable, got %v", err)
	}
}
