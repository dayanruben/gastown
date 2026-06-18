package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	beadsdk "github.com/steveyegge/beads"
	"github.com/steveyegge/gastown/internal/beads"
)

var (
	addTrackingRelationFn    = addTrackingRelation
	removeTrackingRelationFn = removeTrackingRelation
)

func addTrackingRelation(townRoot, trackerID, issueID string) error {
	if err := mutateTrackingRelationViaStore(townRoot, trackerID, issueID, true); err != nil {
		return fallbackTrackingRelation(townRoot, trackerID, issueID, true, err)
	}
	return nil
}

func removeTrackingRelation(townRoot, trackerID, issueID string) error {
	if err := mutateTrackingRelationViaStore(townRoot, trackerID, issueID, false); err != nil {
		return fallbackTrackingRelation(townRoot, trackerID, issueID, false, err)
	}
	return nil
}

func mutateTrackingRelationViaStore(townRoot, trackerID, issueID string, add bool) error {
	resolvedBeads := beads.ResolveBeadsDir(townRoot)
	if resolvedBeads == "" {
		return fmt.Errorf("resolving town beads dir")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	b := beads.NewWithBeadsDir(townRoot, resolvedBeads)
	store, cleanup, err := b.OpenStore(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	targetID := trackingDependsOnID(townRoot, issueID)
	actor := os.Getenv("BD_ACTOR")
	if actor == "" {
		actor = detectSender()
	}

	if add {
		dep := &beadsdk.Dependency{
			IssueID:     trackerID,
			DependsOnID: targetID,
			Type:        beadsdk.DependencyType("tracks"),
		}
		return store.AddDependency(ctx, dep, actor)
	}

	return store.RemoveDependency(ctx, trackerID, targetID, actor)
}

func fallbackTrackingRelation(townRoot, trackerID, issueID string, add bool, storeErr error) error {
	targetID := trackingDependsOnID(townRoot, issueID)
	sqlErr := mutateTrackingRelationViaSQL(townRoot, trackerID, targetID, add)
	if sqlErr == nil {
		return nil
	}

	args := []string{"dep", "add", trackerID, targetID, "--type=tracks"}
	if !add {
		args = []string{"dep", "remove", trackerID, targetID, "--type=tracks"}
	}

	if out, err := BdCmd(args...).Dir(townRoot).WithAutoCommit().StripBeadsDir().CombinedOutput(); err != nil {
		output := strings.TrimSpace(string(out))
		if output == "" {
			return fmt.Errorf("tracking relation via store failed: %w; fallback sql path failed: %v; fallback bd path failed: %w", storeErr, sqlErr, err)
		}
		return fmt.Errorf("tracking relation via store failed: %w; fallback sql path failed: %v; fallback bd path failed: %w; output: %s", storeErr, sqlErr, err, output)
	}

	return nil
}

func mutateTrackingRelationViaSQL(townRoot, trackerID, targetID string, add bool) error {
	if !isValidBeadID(trackerID) {
		return fmt.Errorf("invalid tracker ID: %q", trackerID)
	}
	if !isValidTrackingTargetID(targetID) {
		return fmt.Errorf("invalid tracking target ID: %q", targetID)
	}

	targetColumn := "depends_on_issue_id"
	if strings.HasPrefix(targetID, "external:") {
		targetColumn = "depends_on_external"
	}

	target := sqlStringLiteral(targetID)
	tracker := sqlStringLiteral(trackerID)
	query := fmt.Sprintf(
		"INSERT IGNORE INTO dependencies (issue_id, %s, type, created_at, created_by, metadata) VALUES (%s, %s, 'tracks', NOW(6), %s, '{}')",
		targetColumn,
		tracker,
		target,
		sqlStringLiteral(trackingRelationActor()),
	)
	if !add {
		query = fmt.Sprintf(
			"DELETE FROM dependencies WHERE issue_id = %s AND COALESCE(depends_on_issue_id, depends_on_wisp_id, depends_on_external) = %s AND type = 'tracks'",
			tracker,
			target,
		)
	}

	if out, err := BdCmd("sql", query).Dir(townRoot).WithAutoCommit().StripBeadsDir().CombinedOutput(); err != nil {
		output := strings.TrimSpace(string(out))
		if output == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, output)
	}
	return nil
}

func trackingRelationActor() string {
	if actor := os.Getenv("BD_ACTOR"); actor != "" {
		return actor
	}
	return detectSender()
}

func isValidTrackingTargetID(id string) bool {
	if strings.HasPrefix(id, "external:") {
		parts := strings.Split(id, ":")
		if len(parts) != 3 {
			return false
		}
		return isValidExternalRefPart(parts[1]) && isValidBeadID(parts[2])
	}
	return isValidBeadID(id)
}

func isValidExternalRefPart(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func sqlStringLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func trackingDependsOnID(townRoot, issueID string) string {
	if strings.HasPrefix(issueID, "external:") {
		return issueID
	}

	prefix := beads.ExtractPrefix(issueID)
	if prefix == "" {
		return issueID
	}

	if rigName := beads.GetRigNameForPrefix(townRoot, prefix); rigName != "" {
		return fmt.Sprintf("external:%s:%s", strings.TrimSuffix(prefix, "-"), issueID)
	}

	return issueID
}
