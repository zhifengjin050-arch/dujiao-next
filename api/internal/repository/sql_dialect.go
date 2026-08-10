package repository

import (
	"fmt"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"

	"gorm.io/gorm"
)

var localizedJSONSearchKeys = append([]string(nil), constants.SupportedLocales...)

// dbDialectName ?????????,??? sqlite ???
func dbDialectName(db *gorm.DB) string {
	if db == nil || db.Dialector == nil {
		return "sqlite"
	}
	name := strings.ToLower(strings.TrimSpace(db.Dialector.Name()))
	if name == "" {
		return "sqlite"
	}
	return name
}

// jsonTextExpr ?? JSON ?????????,?? sqlite ? postgres?
func jsonTextExpr(db *gorm.DB, column, key string) string {
	return jsonTextExprByDialect(dbDialectName(db), column, key)
}

func jsonTextExprByDialect(dialect, column, key string) string {
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "postgres", "postgresql":
		// postgres ??? jsonb ???? ->> ????
		return fmt.Sprintf("(%s::jsonb ->> '%s')", column, key)
	default:
		// sqlite ?? json_extract,????????? - ???????
		return fmt.Sprintf("json_extract(%s, '$.\"%s\"')", column, key)
	}
}

// localizedJSONCoalesceExpr ?????????????
func localizedJSONCoalesceExpr(db *gorm.DB, column string) string {
	parts := make([]string, 0, len(localizedJSONSearchKeys)+1)
	for _, key := range localizedJSONSearchKeys {
		parts = append(parts, jsonTextExpr(db, column, key))
	}
	parts = append(parts, "''")
	return fmt.Sprintf("COALESCE(%s)", strings.Join(parts, ", "))
}

// buildLocalizedLikeCondition ????? + JSON ????? LIKE ??,????????
func buildLocalizedLikeCondition(db *gorm.DB, plainColumns, jsonColumns []string) (string, int) {
	return buildLocalizedLikeConditionByDialect(dbDialectName(db), plainColumns, jsonColumns)
}

func buildLocalizedLikeConditionByDialect(dialect string, plainColumns, jsonColumns []string) (string, int) {
	parts := make([]string, 0, len(plainColumns)+len(jsonColumns)*len(localizedJSONSearchKeys))
	argCount := 0
	operator := likeOperatorByDialect(dialect)

	for _, column := range plainColumns {
		trimmed := strings.TrimSpace(column)
		if trimmed == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %s ?", trimmed, operator))
		argCount++
	}

	for _, column := range jsonColumns {
		trimmed := strings.TrimSpace(column)
		if trimmed == "" {
			continue
		}
		for _, key := range localizedJSONSearchKeys {
			parts = append(parts, fmt.Sprintf("%s %s ?", jsonTextExprByDialect(dialect, trimmed, key), operator))
			argCount++
		}
	}

	return strings.Join(parts, " OR "), argCount
}

func likeOperatorByDialect(dialect string) string {
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "postgres", "postgresql":
		return "ILIKE"
	default:
		return "LIKE"
	}
}

// repeatLikeArgs ????? LIKE ?????
func repeatLikeArgs(like string, count int) []interface{} {
	args := make([]interface{}, 0, count)
	for i := 0; i < count; i++ {
		args = append(args, like)
	}
	return args
}

// dateGroupExpr ?? SQL ???????,? UTC ?????????? YYYY-MM-DD ????
// refTime ?? SQLite ???? UTC ??(SQLite ???????)?
func dateGroupExpr(db *gorm.DB, column string, loc *time.Location, refTime time.Time) string {
	if loc == nil {
		loc = time.UTC
	}
	dialect := dbDialectName(db)
	switch dialect {
	case "postgres", "postgresql":
		zoneName := loc.String()
		if zoneName == "" || zoneName == "Local" {
			zoneName = "UTC"
		}
		return fmt.Sprintf("TO_CHAR(%s AT TIME ZONE '%s', 'YYYY-MM-DD')", column, zoneName)
	default: // sqlite
		_, offset := refTime.In(loc).Zone()
		sign := "+"
		if offset < 0 {
			sign = "-"
			offset = -offset
		}
		hours := offset / 3600
		minutes := (offset % 3600) / 60
		if minutes != 0 {
			return fmt.Sprintf("strftime('%%Y-%%m-%%d', %s, '%s%d hours', '%s%d minutes')", column, sign, hours, sign, minutes)
		}
		return fmt.Sprintf("strftime('%%Y-%%m-%%d', %s, '%s%d hours')", column, sign, hours)
	}
}

// quotedStatusList ?????????? SQL IN ???????????????
// ????????,?????????
func quotedStatusList(statuses []string) string {
	parts := make([]string, len(statuses))
	for i, s := range statuses {
		parts[i] = "'" + s + "'"
	}
	return strings.Join(parts, ",")
}
