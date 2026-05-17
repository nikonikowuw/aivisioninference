package scopes

import (
	"reflect"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

var validFieldName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

type Scope = func(*gorm.DB) *gorm.DB

func Paginate(page, pageSize int) Scope {
	return func(db *gorm.DB) *gorm.DB {
		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

// OrderBy returns a Scope that applies ORDER BY with field whitelist validation.
// If sort is empty or not in allowedFields, no ordering is applied.
func OrderBy(sort, order string, allowedFields ...string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		if sort == "" || !validFieldName.MatchString(sort) {
			return db
		}
		for _, f := range allowedFields {
			if sort == f {
				dir := "ASC"
				if strings.ToLower(order) == "desc" {
					dir = "DESC"
				}
				return db.Order(sort + " " + dir)
			}
		}
		return db
	}
}

func Eq(field string, value interface{}) Scope {
	return func(db *gorm.DB) *gorm.DB {
		if value == nil {
			return db.Where(field + " IS NULL")
		}
		v := reflect.ValueOf(value)
		switch v.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
			if v.IsNil() {
				return db.Where(field + " IS NULL")
			}
		}
		return db.Where(field+" = ?", value)
	}
}

func Like(field, value string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(field+" LIKE ?", "%"+value+"%")
	}
}

// MultiLike generates an OR-combined LIKE condition across multiple fields.
// Each field name is validated to prevent SQL injection.
func MultiLike(fields []string, value string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		conds := make([]string, 0, len(fields))
		args := make([]interface{}, 0, len(fields))
		for _, f := range fields {
			if !validFieldName.MatchString(f) {
				continue
			}
			conds = append(conds, f+" LIKE ?")
			args = append(args, "%"+value+"%")
		}
		if len(conds) == 0 {
			return db
		}
		return db.Where(strings.Join(conds, " OR "), args...)
	}
}

func TimeRange(field string, from, to *time.Time) Scope {
	return func(db *gorm.DB) *gorm.DB {
		if from != nil {
			db = db.Where(field+" >= ?", *from)
		}
		if to != nil {
			db = db.Where(field+" <= ?", *to)
		}
		return db
	}
}
