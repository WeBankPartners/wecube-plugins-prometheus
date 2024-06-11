package monitor

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/WeBankPartners/go-common-lib/guid"
	"github.com/WeBankPartners/open-monitor/monitor-server/middleware"
	"github.com/WeBankPartners/open-monitor/monitor-server/models"
	"github.com/WeBankPartners/open-monitor/monitor-server/services/db"
	"github.com/gin-gonic/gin"
)

// GetAllCustomDashboardList 获取所有看板(包括源看板、应用看板)
func GetAllCustomDashboardList(c *gin.Context) {
	var list []*models.SimpleCustomDashboardDto
	var err error
	if list, err = db.QueryAllCustomDashboard(); err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	middleware.ReturnSuccessData(c, list)
}

// QueryCustomDashboardList 查询自定义看板列表
func QueryCustomDashboardList(c *gin.Context) {
	var param models.CustomDashboardQueryParam
	var err error
	var pageInfo models.PageInfo
	var rowsData []*models.CustomDashboardResultDto
	var list []*models.CustomDashboardTable
	var roleRelList []*models.CustomDashBoardRoleRel
	var mainDashBoardList []*models.MainDashboard
	var mgmtRoles, displayMgmtRoles, useRoles, displayUseRoles, mainPages []string
	var displayNameRoleMap map[string]string
	var userRoleMap map[string]bool
	var permission string
	if err = c.ShouldBindJSON(&param); err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	if param.PageSize == 0 {
		param.PageSize = 10
	}
	if pageInfo, list, err = db.QueryCustomDashboardList(param, middleware.GetOperateUserRoles(c)); err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	if displayNameRoleMap, err = db.QueryAllRoleDisplayNameMap(); err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	userRoleMap = db.TransformArrayToMap(middleware.GetOperateUserRoles(c))
	if len(list) > 0 {
		for _, dashboard := range list {
			mgmtRoles = []string{}
			useRoles = []string{}
			displayMgmtRoles = []string{}
			displayUseRoles = []string{}
			mainPages = []string{}
			permission = string(models.PermissionUse)
			if roleRelList, err = db.QueryCustomDashboardRoleRelByCustomDashboard(dashboard.Id); err != nil {
				middleware.ReturnServerHandleError(c, err)
				return
			}
			if mainDashBoardList, err = db.QueryMainDashboardByCustomDashboard(dashboard.Id); err != nil {
				middleware.ReturnServerHandleError(c, err)
				return
			}
			if len(roleRelList) > 0 {
				for _, roleRel := range roleRelList {
					if roleRel.Permission == string(models.PermissionMgmt) {
						mgmtRoles = append(mgmtRoles, roleRel.RoleId)
						if v, ok := displayNameRoleMap[roleRel.RoleId]; ok {
							displayMgmtRoles = append(displayMgmtRoles, v)
						}
						if userRoleMap[roleRel.RoleId] {
							permission = string(models.PermissionMgmt)
						}
					} else if roleRel.Permission == string(models.PermissionUse) {
						useRoles = append(useRoles, roleRel.RoleId)
						if v, ok := displayNameRoleMap[roleRel.RoleId]; ok {
							displayUseRoles = append(displayUseRoles, v)
						}
					}
				}
			}
			if len(mainDashBoardList) > 0 {
				for _, mainDashBoard := range mainDashBoardList {
					if v, ok := displayNameRoleMap[mainDashBoard.RoleId]; ok {
						mainPages = append(mainPages, v)
					}
				}
			}
			result := &models.CustomDashboardResultDto{
				Id:               dashboard.Id,
				Name:             dashboard.Name,
				MgmtRoles:        mgmtRoles,
				DisplayMgmtRoles: displayMgmtRoles,
				UseRoles:         useRoles,
				DisplayUseRoles:  displayUseRoles,
				Permission:       permission,
				CreateUser:       dashboard.CreateUser,
				UpdateUser:       dashboard.UpdateUser,
				UpdateTime:       dashboard.UpdateAt.Format(models.DatetimeFormat),
				MainPage:         mainPages,
			}
			rowsData = append(rowsData, result)
		}
	}
	middleware.ReturnPageData(c, pageInfo, rowsData)
}

func GetCustomDashboard(c *gin.Context) {
	var err error
	var customDashboard *models.CustomDashboardTable
	var customDashboardDto = &models.CustomDashboardDto{UseRoles: []string{}, MgmtRoles: []string{}}
	var customChartExtendList []*models.CustomChartExtend
	var groupMap = make(map[string]bool)
	var configMap = make(map[string][]*models.CustomChartSeriesConfig)
	var tagMap = make(map[string][]*models.CustomChartSeriesTag)
	var tagValueMap = make(map[string][]*models.CustomChartSeriesTagValue)
	var boardRoleRelList []*models.CustomDashBoardRoleRel
	id, _ := strconv.Atoi(c.Query("id"))
	if id == 0 {
		middleware.ReturnParamEmptyError(c, "id")
		return
	}
	// 获取自定义看板
	if customDashboard, err = db.GetCustomDashboardById(id); err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	if customDashboard == nil || customDashboard.Id == 0 {
		middleware.ReturnValidateError(c, "id is invalid")
		return
	}
	customDashboardDto.Name = customDashboard.Name
	customDashboardDto.TimeRange = customDashboard.TimeRange
	customDashboardDto.RefreshWeek = customDashboard.RefreshWeek
	if customChartExtendList, err = db.QueryCustomChartListByDashboard(customDashboard.Id); err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	if configMap, err = db.QueryAllChartSeriesConfig(); err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	if tagMap, err = db.QueryAllChartSeriesTag(); err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	if tagValueMap, err = db.QueryAllChartSeriesTagValue(); err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	if len(customChartExtendList) > 0 {
		customDashboardDto.Charts = []*models.CustomChartDto{}
		for _, chartExtend := range customChartExtendList {
			groupMap[chartExtend.Group] = true
			chart, err2 := db.CreateCustomChartDto(chartExtend, configMap, tagMap, tagValueMap)
			if err2 != nil {
				middleware.ReturnServerHandleError(c, err)
				return
			}
			if chart != nil {
				customDashboardDto.Charts = append(customDashboardDto.Charts, chart)
			}
		}
	}
	if strings.TrimSpace(customDashboard.PanelGroups) == "" {
		customDashboardDto.PanelGroupList = db.TransformMapToArray(groupMap)
	} else {
		customDashboardDto.PanelGroupList = strings.Split(customDashboard.PanelGroups, ",")
	}
	if boardRoleRelList, err = db.QueryCustomDashboardPermissionByDashboard(id); err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	if len(boardRoleRelList) > 0 {
		for _, rel := range boardRoleRelList {
			if rel.Permission == string(models.PermissionUse) {
				customDashboardDto.UseRoles = append(customDashboardDto.UseRoles, rel.RoleId)
			} else if rel.Permission == string(models.PermissionMgmt) {
				customDashboardDto.MgmtRoles = append(customDashboardDto.MgmtRoles, rel.RoleId)
			}
		}
	}
	middleware.ReturnSuccessData(c, customDashboardDto)
}

// AddCustomDashboard 新增自定义看板
func AddCustomDashboard(c *gin.Context) {
	var err error
	var param models.AddCustomDashboardParam
	var dashboardId int64
	if err = c.ShouldBindJSON(&param); err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	if strings.TrimSpace(param.Name) == "" {
		middleware.ReturnParamEmptyError(c, "name")
		return
	}
	if len(param.MgmtRoles) != 1 {
		middleware.ReturnValidateError(c, "mgmtRoles error")
		return
	}
	if len(param.UseRoles) == 0 {
		middleware.ReturnParamEmptyError(c, "useRoles")
		return
	}
	now := time.Now()
	user := middleware.GetOperateUser(c)
	dashboard := &models.CustomDashboardTable{
		Name:       param.Name,
		CreateUser: user,
		UpdateUser: user,
		CreateAt:   now,
		UpdateAt:   now,
	}
	if dashboardId, err = db.AddCustomDashboard(dashboard, param.MgmtRoles, param.UseRoles); err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	dashboard.Id = int(dashboardId)
	middleware.ReturnSuccessData(c, dashboard)
}

// DeleteCustomDashboard 删除自定义看板
func DeleteCustomDashboard(c *gin.Context) {
	var err error
	var id int
	var permission bool
	id, err = strconv.Atoi(c.Query("id"))
	if err != nil || id <= 0 {
		middleware.ReturnParamTypeError(c, "id", "int")
		return
	}
	if permission, err = CheckHasDashboardManagePermission(id, middleware.GetOperateUserRoles(c), middleware.GetOperateUser(c)); err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	if !permission {
		middleware.ReturnServerHandleError(c, fmt.Errorf("no delete permission"))
	}
	if err = db.DeleteCustomDashboardById(id); err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	middleware.ReturnSuccess(c)
}

// UpdateCustomDashboard 修改自定义看板
func UpdateCustomDashboard(c *gin.Context) {
	var err error
	var param models.UpdateCustomDashboardParam
	var hasChartRelList, insertChartRelList, updateChartRelList []*models.CustomDashboardChartRel
	var deleteChartRelIds []string
	var insert, delete, permission bool
	var actions []*db.Action
	var nameMap = make(map[string]bool)
	var panelGroups string
	user := middleware.GetOperateUser(c)
	now := time.Now().Format(models.DatetimeFormat)
	if err = c.ShouldBindJSON(&param); err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	if param.Id == 0 {
		middleware.ReturnParamEmptyError(c, "id")
		return
	}
	if strings.TrimSpace(param.Name) == "" {
		middleware.ReturnParamEmptyError(c, "name")
		return
	}
	if permission, err = CheckHasDashboardManagePermission(param.Id, middleware.GetOperateUserRoles(c), middleware.GetOperateUser(c)); err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	if !permission {
		middleware.ReturnServerHandleError(c, fmt.Errorf("no edit permission"))
		return
	}
	if len(param.Charts) > 0 {
		for _, chart := range param.Charts {
			if nameMap[chart.Name] {
				middleware.ReturnValidateError(c, fmt.Sprintf("chart name:%s repeat", chart.Name))
				return
			}
			nameMap[chart.Name] = true
		}
	}
	if hasChartRelList, err = db.QueryCustomDashboardChartRelListByDashboard(param.Id); err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	if len(param.Charts) == 0 {
		param.Charts = []*models.CustomChartDto{}
	}
	if len(hasChartRelList) == 0 {
		hasChartRelList = []*models.CustomDashboardChartRel{}
	}

	for _, chart := range param.Charts {
		insert = true
		for _, chartRel := range hasChartRelList {
			if *chartRel.DashboardChart == chart.Id {
				displayConfig, _ := json.Marshal(chart.DisplayConfig)
				updateChartRelList = append(updateChartRelList, &models.CustomDashboardChartRel{
					Guid:          chartRel.Guid,
					Group:         chart.Group,
					DisplayConfig: string(displayConfig),
				})
				insert = false
				break
			}
		}
		if insert {
			displayConfig, _ := json.Marshal(chart.DisplayConfig)
			insertChartRelList = append(insertChartRelList, &models.CustomDashboardChartRel{
				Guid:            guid.CreateGuid(),
				CustomDashboard: &param.Id,
				DashboardChart:  &chart.Id,
				Group:           chart.Group,
				DisplayConfig:   string(displayConfig),
				CreateUser:      user,
				UpdateUser:      user,
				CreateTime:      now,
				UpdateTime:      now,
			})
		}
	}

	for _, chartRel := range hasChartRelList {
		delete = true
		for _, chart := range param.Charts {
			if chart.Id == *chartRel.DashboardChart {
				delete = false
				break
			}
		}
		if delete {
			deleteChartRelIds = append(deleteChartRelIds, chartRel.Guid)
		}
	}
	if len(insertChartRelList) > 0 {
		actions = append(actions, db.GetAddCustomDashboardChartRelSQL(insertChartRelList)...)
	}
	if len(updateChartRelList) > 0 {
		actions = append(actions, db.GetUpdateCustomDashboardChartRelSQL(updateChartRelList)...)
	}
	if len(deleteChartRelIds) > 0 {
		actions = append(actions, db.GetDeleteCustomDashboardChartRelSQL(deleteChartRelIds)...)
	}
	if len(param.PanelGroups) > 0 {
		panelGroups = strings.Join(param.PanelGroups, ",")
	}
	actions = append(actions, db.GetUpdateCustomDashboardSQL(param.Name, panelGroups, middleware.GetOperateUser(c), param.TimeRange, param.RefreshWeek, param.Id)...)
	if err = db.Transaction(actions); err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	middleware.ReturnSuccess(c)
}

// UpdateCustomDashboardPermission 修改自定义面板权限
func UpdateCustomDashboardPermission(c *gin.Context) {
	var err error
	var param models.UpdateCustomDashboardPermissionParam
	var actions, deleteActions, subActions, updateActions []*db.Action
	var permission bool
	if err = c.ShouldBindJSON(&param); err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	if len(param.MgmtRoles) != 1 {
		middleware.ReturnValidateError(c, "mgmtRoles error")
		return
	}
	if len(param.UseRoles) == 0 {
		middleware.ReturnParamEmptyError(c, "useRoles is empty")
		return
	}
	if permission, err = CheckHasDashboardManagePermission(param.Id, middleware.GetOperateUserRoles(c), middleware.GetOperateUser(c)); err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	if !permission {
		middleware.ReturnServerHandleError(c, fmt.Errorf("not has edit permission"))
	}
	deleteActions = db.GetDeleteCustomDashboardRoleRelSQL(param.Id)
	if len(deleteActions) > 0 {
		actions = append(actions, deleteActions...)
	}
	subActions = db.GetInsertCustomDashboardRoleRelSQL(param.Id, param.MgmtRoles, param.UseRoles)
	if len(subActions) > 0 {
		actions = append(actions, subActions...)
	}
	updateActions = db.UpdateCustomDashboardTimeActions(param.Id, middleware.GetOperateUser(c))
	actions = append(actions, updateActions...)
	if err = db.Transaction(actions); err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	middleware.ReturnSuccess(c)
}

func CheckHasDashboardManagePermission(dashboard int, userRoles []string, user string) (permission bool, err error) {
	var permissionMap map[string]string
	var customDashboard *models.CustomDashboardTable
	if len(userRoles) == 0 {
		return
	}
	// 判断是否拥有删除权限
	if permissionMap, err = db.QueryCustomDashboardManagePermissionByDashboard(dashboard); err != nil {
		return
	}
	if len(permissionMap) == 0 {
		permissionMap = make(map[string]string)
	}
	for _, role := range userRoles {
		if v, ok := permissionMap[role]; ok && v == string(models.PermissionMgmt) {
			permission = true
			break
		}
	}
	if !permission && user != "" {
		if customDashboard, err = db.GetCustomDashboardById(dashboard); err != nil {
			return
		}
		if customDashboard != nil && user == customDashboard.CreateUser {
			permission = true
			return
		}
	}
	return
}

func SyncData(c *gin.Context) {
	err := db.SyncData()
	if err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	middleware.ReturnSuccess(c)
}

// UnBindChart 解除图表绑定,看板不删除,私有图表删除
func UnBindChart(c *gin.Context) {
	dashboardId, _ := strconv.Atoi(c.Query("dashboard_id"))
	if dashboardId == 0 {
		middleware.ReturnValidateError(c, "dashboard_id is invalid")
		return
	}
	err := db.UnBindChart(dashboardId)
	if err != nil {
		middleware.ReturnServerHandleError(c, err)
		return
	}
	middleware.ReturnSuccess(c)
}
