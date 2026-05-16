package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
)

func TestToRoleInfosAndIDs(t *testing.T) {
	roles := []model.Role{
		{
			BaseModel:   model.BaseModel{ID: "role-1"},
			Name:        "Admin",
			Description: "Admin role",
		},
		{
			BaseModel:   model.BaseModel{ID: "role-2"},
			Name:        "User",
			Description: "User role",
		},
	}

	infos, ids := toRoleInfosAndIDs(roles)

	assert.Len(t, infos, 2)
	assert.Len(t, ids, 2)
	assert.Equal(t, "role-1", ids[0])
	assert.Equal(t, "role-2", ids[1])
	assert.Equal(t, "Admin", infos[0].Name)
	assert.Equal(t, "User", infos[1].Name)
}

func TestBuildMenuNodes(t *testing.T) {
	parentID := "menu-root"
	perms := []model.Permission{
		{BaseModel: model.BaseModel{ID: "menu-1"}, Name: "A", Code: "a", Path: "/a", Icon: "i-a", SortOrder: 20, ParentID: &parentID},
		{BaseModel: model.BaseModel{ID: "menu-2"}, Name: "B", Code: "b", Path: "/b", Icon: "i-b", SortOrder: 10},
		{BaseModel: model.BaseModel{ID: "menu-1"}, Name: "A-dup", Code: "a-dup", Path: "/a-dup", Icon: "i-a-dup", SortOrder: 99},
	}

	nodes := buildMenuNodes(perms)

	assert.Len(t, nodes, 2)
	assert.Equal(t, "menu-root", nodes["menu-1"].parentID)
	assert.Equal(t, "A", nodes["menu-1"].menu.Name)
	assert.Equal(t, 20, nodes["menu-1"].menu.SortOrder)
	assert.Equal(t, "", nodes["menu-2"].parentID)
}

func TestBuildMenuRelations(t *testing.T) {
	tests := []struct {
		name                 string
		nodes                map[string]*menuNode
		expectRootIDs        []string
		expectChildrenByRoot map[string][]string
	}{
		{
			name: "normal and orphan parent",
			nodes: map[string]*menuNode{
				"root":   {menu: dto.Menu{ID: "root", SortOrder: 1}, parentID: ""},
				"child1": {menu: dto.Menu{ID: "child1", SortOrder: 2}, parentID: "root"},
				"child2": {menu: dto.Menu{ID: "child2", SortOrder: 3}, parentID: "root"},
				"orphan": {menu: dto.Menu{ID: "orphan", SortOrder: 4}, parentID: "not-exist"},
			},
			expectRootIDs: []string{"root", "orphan"},
			expectChildrenByRoot: map[string][]string{
				"root": {"child1", "child2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			childrenByParent, rootIDs := buildMenuRelations(tt.nodes)
			assert.ElementsMatch(t, tt.expectRootIDs, rootIDs)
			assert.Len(t, childrenByParent, len(tt.expectChildrenByRoot))
			for k, v := range tt.expectChildrenByRoot {
				assert.ElementsMatch(t, v, childrenByParent[k])
			}
		})
	}
}

func TestBuildMenuTree(t *testing.T) {
	tests := []struct {
		name            string
		perms           []model.Permission
		expectRootOrder []string
		expectChildren  map[string][]string
	}{
		{
			name: "sort root and children by sort_order",
			perms: []model.Permission{
				{BaseModel: model.BaseModel{ID: "root-b"}, Name: "B", SortOrder: 20},
				{BaseModel: model.BaseModel{ID: "root-a"}, Name: "A", SortOrder: 10},
				{BaseModel: model.BaseModel{ID: "child-a2"}, Name: "A2", SortOrder: 20, ParentID: strPtr("root-a")},
				{BaseModel: model.BaseModel{ID: "child-a1"}, Name: "A1", SortOrder: 10, ParentID: strPtr("root-a")},
				{BaseModel: model.BaseModel{ID: "orphan"}, Name: "Orphan", SortOrder: 15, ParentID: strPtr("missing")},
			},
			expectRootOrder: []string{"root-a", "orphan", "root-b"},
			expectChildren: map[string][]string{
				"root-a": {"child-a1", "child-a2"},
				"orphan": {},
				"root-b": {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			menus := buildMenuTree(tt.perms)
			assert.Equal(t, tt.expectRootOrder, extractIDs(menus))
			for _, m := range menus {
				expectChildIDs, ok := tt.expectChildren[m.ID]
				if !ok {
					continue
				}
				assert.Equal(t, expectChildIDs, extractIDs(m.Children))
			}
		})
	}
}

func strPtr(v string) *string {
	return &v
}

func extractIDs(menus []dto.Menu) []string {
	ids := make([]string, 0, len(menus))
	for _, m := range menus {
		ids = append(ids, m.ID)
	}
	return ids
}
