package db

import (
	"encoding/json"
	"fmt"
	"github.com/WeBankPartners/go-common-lib/guid"
	"github.com/WeBankPartners/go-common-lib/pcre"
	"github.com/WeBankPartners/open-monitor/monitor-server/middleware/log"
	"github.com/WeBankPartners/open-monitor/monitor-server/models"
	"github.com/dlclark/regexp2"
	"strings"
	"time"
)

func GetLogMetricByServiceGroup(serviceGroup string) (result models.LogMetricQueryObj, err error) {
	serviceGroupObj, getErr := getSimpleServiceGroup(serviceGroup)
	if getErr != nil {
		return result, getErr
	}
	result.ServiceGroupTable = serviceGroupObj
	result.Config = []*models.LogMetricMonitorObj{}
	var logMetricMonitorTable []*models.LogMetricMonitorTable
	err = x.SQL("select * from log_metric_monitor where service_group=?", serviceGroup).Find(&logMetricMonitorTable)
	if err != nil {
		return
	}
	for _, logMetricMonitor := range logMetricMonitorTable {
		tmpConfig := models.LogMetricMonitorObj{Guid: logMetricMonitor.Guid, ServiceGroup: logMetricMonitor.ServiceGroup, LogPath: logMetricMonitor.LogPath, MetricType: logMetricMonitor.MetricType, MonitorType: logMetricMonitor.MonitorType}
		tmpConfig.EndpointRel = ListLogMetricEndpointRel(logMetricMonitor.Guid)
		tmpConfig.JsonConfigList = ListLogMetricJson(logMetricMonitor.Guid)
		tmpConfig.MetricConfigList = ListLogMetricConfig("", logMetricMonitor.Guid)
		tmpConfig.MetricGroups = ListLogMetricGroups(logMetricMonitor.Guid)
		for _, logJsonObj := range tmpConfig.JsonConfigList {
			for _, logMetricObj := range logJsonObj.MetricList {
				logMetricObj.ServiceGroup = serviceGroup
				logMetricObj.MonitorType = logMetricMonitor.MonitorType
			}
		}
		for _, logMetricObj := range tmpConfig.MetricConfigList {
			logMetricObj.ServiceGroup = serviceGroup
			logMetricObj.MonitorType = logMetricMonitor.MonitorType
		}
		result.Config = append(result.Config, &tmpConfig)
	}
	return
}

func GetLogMetricByEndpoint(endpoint string, onlySource bool) (result []*models.LogMetricQueryObj, err error) {
	result = []*models.LogMetricQueryObj{}
	var endpointServiceRelTable []*models.EndpointServiceRelTable
	if onlySource {
		err = x.SQL("select distinct t2.service_group from log_metric_endpoint_rel t1 left join log_metric_monitor t2 on t1.log_metric_monitor=t2.guid where t1.source_endpoint=?", endpoint).Find(&endpointServiceRelTable)
	} else {
		err = x.SQL("select distinct t2.service_group from log_metric_endpoint_rel t1 left join log_metric_monitor t2 on t1.log_metric_monitor=t2.guid where t1.source_endpoint=? or t1.target_endpoint=?", endpoint, endpoint).Find(&endpointServiceRelTable)
	}
	for _, v := range endpointServiceRelTable {
		tmpObj, tmpErr := GetLogMetricByServiceGroup(v.ServiceGroup)
		if tmpErr != nil {
			err = tmpErr
			break
		}
		result = append(result, &tmpObj)
	}
	return
}

func ListLogMetricEndpointRel(logMetricMonitor string) (result []*models.LogMetricEndpointRelTable) {
	result = []*models.LogMetricEndpointRelTable{}
	x.SQL("select * from log_metric_endpoint_rel where log_metric_monitor=?", logMetricMonitor).Find(&result)
	return result
}

func ListLogMetricEndpointRelWithServiceGroup(serviceGroup, logMetricMonitor string) (result []*models.LogMetricEndpointRelTable) {
	result = []*models.LogMetricEndpointRelTable{}
	if serviceGroup == "" {
		var logMetricMonitorTable []*models.LogMetricMonitorTable
		x.SQL("select service_group from log_metric_monitor where guid=?", logMetricMonitor).Find(&logMetricMonitorTable)
		if len(logMetricMonitorTable) > 0 {
			serviceGroup = logMetricMonitorTable[0].ServiceGroup
		} else {
			return result
		}
	}
	endpointList, _ := ListServiceGroupEndpoint(serviceGroup, "host")
	var logMetricRelTable []*models.LogMetricEndpointRelTable
	x.SQL("select * from log_metric_endpoint_rel where log_metric_monitor=?", logMetricMonitor).Find(&logMetricRelTable)
	endpointRelMap := make(map[string]*models.LogMetricEndpointRelTable)
	for _, v := range logMetricRelTable {
		endpointRelMap[v.SourceEndpoint] = v
	}
	for _, v := range endpointList {
		if existRow, b := endpointRelMap[v.Guid]; b {
			result = append(result, existRow)
		} else {
			result = append(result, &models.LogMetricEndpointRelTable{SourceEndpoint: v.Guid})
		}
	}
	return result
}

func GetServiceGroupEndpointRel(serviceGroup, sourceType, targetType string) (result []*models.LogMetricEndpointRelTable, err error) {
	result = []*models.LogMetricEndpointRelTable{}
	var guidList []string
	guidList, err = fetchGlobalServiceGroupChildGuidList(serviceGroup)
	if err != nil {
		return
	}
	var endpointTable []*models.EndpointNewTable
	err = x.SQL("select guid,monitor_type,ip from endpoint_new where guid in (select endpoint from endpoint_service_rel where service_group in ('" + strings.Join(guidList, "','") + "'))").Find(&endpointTable)
	if err != nil {
		return
	}
	sourceMap := make(map[string]string)
	targetMap := make(map[string]string)
	var tmpResult []*models.LogMetricEndpointRelTable
	for _, v := range endpointTable {
		if v.MonitorType == sourceType {
			sourceMap[v.Guid] = v.Ip
			tmpResult = append(tmpResult, &models.LogMetricEndpointRelTable{SourceEndpoint: v.Guid})
		}
		if v.MonitorType == targetType {
			targetMap[v.Ip] = v.Guid
		}
	}
	for _, v := range tmpResult {
		if targetGuid, b := targetMap[sourceMap[v.SourceEndpoint]]; b {
			v.TargetEndpoint = targetGuid
			result = append(result, v)
		}
	}
	return
}

func ListLogMetricJson(logMetricMonitor string) (result []*models.LogMetricJsonObj) {
	result = []*models.LogMetricJsonObj{}
	var logMetricJsonTable []*models.LogMetricJsonTable
	x.SQL("select * from log_metric_json where log_metric_monitor=?", logMetricMonitor).Find(&logMetricJsonTable)
	for _, v := range logMetricJsonTable {
		result = append(result, &models.LogMetricJsonObj{Guid: v.Guid, LogMetricMonitor: v.LogMetricMonitor, JsonRegular: v.JsonRegular, Tags: v.Tags, MetricList: ListLogMetricConfig(v.Guid, "")})
	}
	return result
}

func ListLogMetricConfig(logMetricJson, logMetricMonitor string) (result []*models.LogMetricConfigObj) {
	result = []*models.LogMetricConfigObj{}
	var logMetricConfigTable []*models.LogMetricConfigTable
	if logMetricJson != "" {
		x.SQL("select * from log_metric_config where log_metric_json=?", logMetricJson).Find(&logMetricConfigTable)
	} else {
		x.SQL("select * from log_metric_config where log_metric_monitor=?", logMetricMonitor).Find(&logMetricConfigTable)
	}
	for _, v := range logMetricConfigTable {
		tmpTagConfig := []*models.LogMetricConfigTag{}
		if v.TagConfig != "" {
			if tmpErr := json.Unmarshal([]byte(v.TagConfig), &tmpTagConfig); tmpErr != nil {
				log.Logger.Warn("query log metric config warning with json unmarshal error", log.String("tagConfig", v.TagConfig), log.Error(tmpErr))
			}
		}
		tmpJsonTagList := []string{}
		for _, tagConfigItem := range tmpTagConfig {
			tmpJsonTagList = append(tmpJsonTagList, tagConfigItem.Key)
		}
		result = append(result, &models.LogMetricConfigObj{Guid: v.Guid, LogMetricMonitor: v.LogMetricMonitor, LogMetricJson: v.LogMetricJson, Metric: v.Metric, DisplayName: v.DisplayName, JsonKey: v.JsonKey, Regular: v.Regular, AggType: v.AggType, Step: v.Step, StringMap: ListLogMetricStringMap(v.Guid), TagConfig: tmpTagConfig, JsonTagList: tmpJsonTagList})
	}
	return result
}

func ListLogMetricStringMap(logMetricConfig string) (result []*models.LogMetricStringMapTable) {
	result = []*models.LogMetricStringMapTable{}
	x.SQL("select * from log_metric_string_map where log_metric_config=?", logMetricConfig).Find(&result)
	return result
}

func CreateLogMetricMonitor(param *models.LogMetricMonitorCreateDto) error {
	if len(param.LogPath) == 0 {
		return nil
	}
	nowTime := time.Now().Format(models.DatetimeFormat)
	var actions []*Action
	logMonitorGuidList := guid.CreateGuidList(len(param.LogPath))
	for i, v := range param.LogPath {
		tmpLogPath := strings.TrimSpace(v)
		actions = append(actions, &Action{Sql: "insert into log_metric_monitor(guid,service_group,log_path,metric_type,monitor_type,update_time) value (?,?,?,?,?,?)", Param: []interface{}{logMonitorGuidList[i], param.ServiceGroup, tmpLogPath, param.MetricType, param.MonitorType, nowTime}})
		relGuidList := guid.CreateGuidList(len(param.EndpointRel))
		for ii, vv := range param.EndpointRel {
			if vv.TargetEndpoint == "" {
				continue
			}
			actions = append(actions, &Action{Sql: "insert into log_metric_endpoint_rel(guid,log_metric_monitor,source_endpoint,target_endpoint) value (?,?,?,?)", Param: []interface{}{relGuidList[ii], logMonitorGuidList[i], vv.SourceEndpoint, vv.TargetEndpoint}})
		}
	}
	return Transaction(actions)
}

func GetLogMetricMonitor(logMetricMonitorGuid string) (result models.LogMetricMonitorObj, err error) {
	var logMetricMonitorTable []*models.LogMetricMonitorTable
	err = x.SQL("select * from log_metric_monitor where guid=?", logMetricMonitorGuid).Find(&logMetricMonitorTable)
	if err != nil {
		return result, fmt.Errorf("Query table log_metric_monitor fail,%s ", err.Error())
	}
	if len(logMetricMonitorTable) == 0 {
		return result, fmt.Errorf("Can not find log_metric_monitor with guid:%s ", logMetricMonitorGuid)
	}
	result = models.LogMetricMonitorObj{Guid: logMetricMonitorTable[0].Guid, ServiceGroup: logMetricMonitorTable[0].ServiceGroup, LogPath: logMetricMonitorTable[0].LogPath, MetricType: logMetricMonitorTable[0].MetricType, MonitorType: logMetricMonitorTable[0].MonitorType}
	result.EndpointRel = ListLogMetricEndpointRel(logMetricMonitorTable[0].Guid)
	result.JsonConfigList = ListLogMetricJson(logMetricMonitorTable[0].Guid)
	result.MetricConfigList = ListLogMetricConfig("", logMetricMonitorTable[0].Guid)
	result.MetricGroups = ListLogMetricGroups(logMetricMonitorTable[0].Guid)
	return result, nil
}

func UpdateLogMetricMonitor(param *models.LogMetricMonitorObj) error {
	nowTime := time.Now().Format(models.DatetimeFormat)
	var actions []*Action
	actions = append(actions, &Action{Sql: "update log_metric_monitor set log_path=?,monitor_type=?,update_time=? where guid=?", Param: []interface{}{param.LogPath, param.MonitorType, nowTime, param.Guid}})
	actions = append(actions, &Action{Sql: "delete from log_metric_endpoint_rel where log_metric_monitor=?", Param: []interface{}{param.Guid}})
	guidList := guid.CreateGuidList(len(param.EndpointRel))
	for i, v := range param.EndpointRel {
		if v.TargetEndpoint == "" {
			continue
		}
		actions = append(actions, &Action{Sql: "insert into log_metric_endpoint_rel(guid,log_metric_monitor,source_endpoint,target_endpoint) value (?,?,?,?)", Param: []interface{}{guidList[i], param.Guid, v.SourceEndpoint, v.TargetEndpoint}})
	}
	return Transaction(actions)
}

func DeleteLogMetricMonitor(logMetricMonitorGuid string) (err error) {
	var logMetricMonitorTable []*models.LogMetricMonitorTable
	err = x.SQL("select * from log_metric_monitor where guid=?", logMetricMonitorGuid).Find(&logMetricMonitorTable)
	if len(logMetricMonitorTable) == 0 {
		return
	}
	actions, affectHost, affectEndpointGroup := getDeleteLogMetricMonitor(logMetricMonitorGuid)
	err = Transaction(actions)
	if err != nil {
		return err
	}
	if len(affectHost) > 0 {
		err = SyncLogMetricExporterConfig(affectHost)
		if err != nil {
			log.Logger.Error("SyncLogMetricExporterConfig fail", log.Error(err))
		}
	}
	if len(affectEndpointGroup) > 0 {
		for _, v := range affectEndpointGroup {
			err = SyncPrometheusRuleFile(v, false)
			if err != nil {
				log.Logger.Error("SyncPrometheusRuleFile fail", log.Error(err))
			}
		}
	}
	return nil
}

func getDeleteLogMetricMonitor(logMetricMonitorGuid string) (actions []*Action, affectHost, affectEndpointGroup []string) {
	endpointRel := ListLogMetricEndpointRel(logMetricMonitorGuid)
	jsonConfigList := ListLogMetricJson(logMetricMonitorGuid)
	metricConfigList := ListLogMetricConfig("", logMetricMonitorGuid)
	for _, v := range endpointRel {
		affectHost = append(affectHost, v.SourceEndpoint)
	}
	actions = append(actions, &Action{Sql: "delete from log_metric_endpoint_rel where log_metric_monitor=?", Param: []interface{}{logMetricMonitorGuid}})
	for _, v := range jsonConfigList {
		for _, vv := range v.MetricList {
			deleteActions, tmpEndpointGroup := getDeleteLogMetricConfigAction(vv.Guid, logMetricMonitorGuid)
			actions = append(actions, deleteActions...)
			affectEndpointGroup = append(affectEndpointGroup, tmpEndpointGroup...)
		}
		actions = append(actions, &Action{Sql: "delete from log_metric_json where guid=?", Param: []interface{}{v.Guid}})
	}
	for _, v := range metricConfigList {
		deleteActions, tmpEndpointGroup := getDeleteLogMetricConfigAction(v.Guid, logMetricMonitorGuid)
		actions = append(actions, deleteActions...)
		affectEndpointGroup = append(affectEndpointGroup, tmpEndpointGroup...)
	}
	actions = append(actions, &Action{Sql: "delete from log_metric_monitor where guid=?", Param: []interface{}{logMetricMonitorGuid}})
	return
}

func GetLogMetricJson(logMetricJsonGuid string) (result models.LogMetricJsonObj, err error) {
	var logMetricJsonTable []*models.LogMetricJsonTable
	err = x.SQL("select * from log_metric_json where guid=?", logMetricJsonGuid).Find(&logMetricJsonTable)
	if err != nil {
		return result, fmt.Errorf("Query log_metric_json table fail,%s ", err.Error())
	}
	if len(logMetricJsonTable) == 0 {
		return result, fmt.Errorf("Can not find log_metric_json with guid:%s ", logMetricJsonGuid)
	}
	result = models.LogMetricJsonObj{Guid: logMetricJsonTable[0].Guid, Name: logMetricJsonTable[0].Name, LogMetricMonitor: logMetricJsonTable[0].LogMetricMonitor, JsonRegular: logMetricJsonTable[0].JsonRegular, Tags: logMetricJsonTable[0].Tags, DemoLog: logMetricJsonTable[0].DemoLog, TrialCalculationResult: []string{}}
	json.Unmarshal([]byte(logMetricJsonTable[0].CalcResult), &result.TrialCalculationResult)
	result.MetricList = ListLogMetricConfig(logMetricJsonGuid, "")
	return
}

func CreateLogMetricJson(param *models.LogMetricJsonObj) error {
	nowTime := time.Now().Format(models.DatetimeFormat)
	var actions []*Action
	param.Guid = guid.CreateGuid()
	calcResultBytes, _ := json.Marshal(param.TrialCalculationResult)
	actions = append(actions, &Action{Sql: "insert into log_metric_json(guid,name,log_metric_monitor,json_regular,tags,demo_log,calc_result,update_time) value (?,?,?,?,?,?,?,?)", Param: []interface{}{param.Guid, param.Name, param.LogMetricMonitor, param.JsonRegular, param.Tags, param.DemoLog, string(calcResultBytes), nowTime}})
	guidList := guid.CreateGuidList(len(param.MetricList))
	for i, v := range param.MetricList {
		v.LogMetricJson = param.Guid
		v.LogMetricMonitor = param.LogMetricMonitor
		v.Guid = guidList[i]
		tmpActions := getCreateLogMetricConfigAction(v, nowTime)
		actions = append(actions, tmpActions...)
	}
	return Transaction(actions)
}

func UpdateLogMetricJson(param *models.LogMetricJsonObj) error {
	if param.LogMetricMonitor == "" {
		logMetricMonitorGuid, err := getLogMetricJsonMonitor(param.Guid)
		if err != nil {
			return err
		}
		param.LogMetricMonitor = logMetricMonitorGuid
	}
	nowTime := time.Now().Format(models.DatetimeFormat)
	var actions []*Action
	calcResultBytes, _ := json.Marshal(param.TrialCalculationResult)
	actions = append(actions, &Action{Sql: "update log_metric_json set name=?,json_regular=?,tags=?,demo_log=?,calc_result=?,update_time=? where guid=?", Param: []interface{}{param.Name, param.JsonRegular, param.Tags, param.DemoLog, string(calcResultBytes), nowTime, param.Guid}})
	var logMetricConfigTable []*models.LogMetricConfigTable
	x.SQL("select * from log_metric_config where log_metric_json=?", param.Guid).Find(&logMetricConfigTable)
	var affectEndpointGroup []string
	for _, v := range param.MetricList {
		v.LogMetricJson = param.Guid
		v.LogMetricMonitor = param.LogMetricMonitor
		if v.Guid == "" {
			actions = append(actions, getCreateLogMetricConfigAction(v, nowTime)...)
			continue
		}
		tmpUpdateActions, tmpEndpointGroup := getUpdateLogMetricConfigAction(v, nowTime)
		actions = append(actions, tmpUpdateActions...)
		affectEndpointGroup = append(affectEndpointGroup, tmpEndpointGroup...)
	}
	for _, v := range logMetricConfigTable {
		existFlag := false
		for _, vv := range param.MetricList {
			if v.Guid == vv.Guid {
				existFlag = true
				break
			}
		}
		if !existFlag {
			deleteActions, tmpEndpointGroup := getDeleteLogMetricConfigAction(v.Guid, param.LogMetricMonitor)
			actions = append(actions, deleteActions...)
			affectEndpointGroup = append(affectEndpointGroup, tmpEndpointGroup...)
		}
	}
	err := Transaction(actions)
	if err == nil && len(affectEndpointGroup) > 0 {
		for _, v := range affectEndpointGroup {
			SyncPrometheusRuleFile(v, false)
		}
	}
	return err
}

func DeleteLogMetricJson(logMetricJsonGuid string) (logMetricMonitorGuid string, err error) {
	logMetricMonitorGuid, err = getLogMetricJsonMonitor(logMetricJsonGuid)
	var actions []*Action
	var logMetricConfigTable []*models.LogMetricConfigTable
	var affectEndpointGroup []string
	x.SQL("select * from log_metric_config where log_metric_json=?", logMetricJsonGuid).Find(&logMetricConfigTable)
	for _, v := range logMetricConfigTable {
		deleteActions, tmpEndpointGroup := getDeleteLogMetricConfigAction(v.Guid, logMetricMonitorGuid)
		actions = append(actions, deleteActions...)
		affectEndpointGroup = append(affectEndpointGroup, tmpEndpointGroup...)
	}
	actions = append(actions, &Action{Sql: "delete from log_metric_json where guid=?", Param: []interface{}{logMetricJsonGuid}})
	err = Transaction(actions)
	if err == nil && len(affectEndpointGroup) > 0 {
		for _, v := range affectEndpointGroup {
			SyncPrometheusRuleFile(v, false)
		}
	}
	return
}

func getLogMetricJsonMonitor(logMetricJsonGuid string) (logMetricMonitorGuid string, err error) {
	var logMetricJsonTable []*models.LogMetricJsonTable
	err = x.SQL("select * from log_metric_json where guid=?", logMetricJsonGuid).Find(&logMetricJsonTable)
	if err != nil {
		return logMetricMonitorGuid, fmt.Errorf("Query log_metric_json fail,%s ", err.Error())
	}
	if len(logMetricJsonTable) == 0 {
		return logMetricMonitorGuid, fmt.Errorf("Can not find log_metric_json with guid:%s ", logMetricJsonGuid)
	}
	logMetricMonitorGuid = logMetricJsonTable[0].LogMetricMonitor
	return logMetricMonitorGuid, nil
}

func GetLogMetricConfig(logMetricConfigGuid string) (result models.LogMetricConfigObj, err error) {
	var logMetricConfigTable []*models.LogMetricConfigTable
	err = x.SQL("select * from log_metric_config where guid=?", logMetricConfigGuid).Find(&logMetricConfigTable)
	if err != nil {
		return result, fmt.Errorf("Query table log_metric_config fail,%s ", err.Error())
	}
	if len(logMetricConfigTable) == 0 {
		return result, fmt.Errorf("Can not find log_metric_config with guid:%s ", logMetricConfigGuid)
	}
	result = models.LogMetricConfigObj{Guid: logMetricConfigGuid, LogMetricMonitor: logMetricConfigTable[0].LogMetricMonitor, LogMetricJson: logMetricConfigTable[0].LogMetricJson, Metric: logMetricConfigTable[0].Metric, DisplayName: logMetricConfigTable[0].DisplayName, JsonKey: logMetricConfigTable[0].JsonKey, Regular: logMetricConfigTable[0].Regular, AggType: logMetricConfigTable[0].AggType, Step: logMetricConfigTable[0].Step}
	result.StringMap = ListLogMetricStringMap(logMetricConfigGuid)
	return
}

func CreateLogMetricConfig(param *models.LogMetricConfigObj) error {
	param.Guid = guid.CreateGuid()
	actions := getCreateLogMetricConfigAction(param, time.Now().Format(models.DatetimeFormat))
	return Transaction(actions)
}

func UpdateLogMetricConfig(param *models.LogMetricConfigObj) error {
	logMetricMonitorGuid, err := getLogMetricConfigMonitor(param.Guid)
	if err != nil {
		return fmt.Errorf("Query table log_metric_config fail,%s ", err.Error())
	}
	param.LogMetricMonitor = logMetricMonitorGuid
	actions, affectEndpointGroup := getUpdateLogMetricConfigAction(param, time.Now().Format(models.DatetimeFormat))
	err = Transaction(actions)
	if err == nil {
		for _, v := range affectEndpointGroup {
			SyncPrometheusRuleFile(v, false)
		}
	}
	return err
}

func DeleteLogMetricConfig(logMetricConfigGuid string) (logMetricMonitorGuid string, err error) {
	logMetricMonitorGuid, err = getLogMetricConfigMonitor(logMetricConfigGuid)
	actions, affectEndpointGroup := getDeleteLogMetricConfigAction(logMetricConfigGuid, logMetricMonitorGuid)
	err = Transaction(actions)
	if err == nil && len(affectEndpointGroup) > 0 {
		for _, v := range affectEndpointGroup {
			SyncPrometheusRuleFile(v, false)
		}
	}
	return
}

func getLogMetricConfigMonitor(logMetricConfigGuid string) (logMetricMonitorGuid string, err error) {
	var logMetricConfigTable []*models.LogMetricConfigTable
	err = x.SQL("select guid,log_metric_monitor from log_metric_config where guid=?", logMetricConfigGuid).Find(&logMetricConfigTable)
	if err != nil {
		return logMetricMonitorGuid, fmt.Errorf("Query log_metric_config fail,%s ", err.Error())
	}
	if len(logMetricConfigTable) == 0 {
		return logMetricMonitorGuid, fmt.Errorf("Can not find log_metric_config with guid:%s ", logMetricConfigGuid)
	}
	logMetricMonitorGuid = logMetricConfigTable[0].LogMetricMonitor
	return logMetricMonitorGuid, err
}

func getCreateLogMetricConfigAction(param *models.LogMetricConfigObj, nowTime string) []*Action {
	var actions []*Action
	if param.Guid == "" {
		param.Guid = guid.CreateGuid()
	}
	tagString := ""
	for _, jsonTagItem := range param.JsonTagList {
		param.TagConfig = append(param.TagConfig, &models.LogMetricConfigTag{Key: jsonTagItem})
	}
	if len(param.TagConfig) > 0 {
		tagBytes, _ := json.Marshal(param.TagConfig)
		tagString = string(tagBytes)
	}
	param.Step = 10
	if param.LogMetricJson != "" {
		actions = append(actions, &Action{Sql: "insert into log_metric_config(guid,log_metric_json,metric,display_name,json_key,regular,agg_type,step,update_time,tag_config) value (?,?,?,?,?,?,?,?,?,?)", Param: []interface{}{param.Guid, param.LogMetricJson, param.Metric, param.DisplayName, param.JsonKey, param.Regular, param.AggType, param.Step, nowTime, tagString}})
	} else {
		actions = append(actions, &Action{Sql: "insert into log_metric_config(guid,log_metric_monitor,metric,display_name,json_key,regular,agg_type,step,update_time,tag_config) value (?,?,?,?,?,?,?,?,?,?)", Param: []interface{}{param.Guid, param.LogMetricMonitor, param.Metric, param.DisplayName, param.JsonKey, param.Regular, param.AggType, param.Step, nowTime, tagString}})
	}
	if param.ServiceGroup == "" || param.MonitorType == "" {
		param.ServiceGroup, param.MonitorType = getLogMetricServiceGroup(param.LogMetricMonitor)
	}
	actions = append(actions, &Action{Sql: "insert into metric(guid,metric,monitor_type,prom_expr,service_group,workspace,update_time) value (?,?,?,?,?,?,?)", Param: []interface{}{fmt.Sprintf("%s__%s", param.Metric, param.ServiceGroup), param.Metric, param.MonitorType, getLogMetricExprByAggType(param.Metric, param.AggType, param.ServiceGroup, []string{}), param.ServiceGroup, models.MetricWorkspaceService, nowTime}})
	guidList := guid.CreateGuidList(len(param.StringMap))
	for i, v := range param.StringMap {
		actions = append(actions, &Action{Sql: "insert into log_metric_string_map(guid,log_metric_config,source_value,regulative,target_value,update_time) value (?,?,?,?,?,?)", Param: []interface{}{guidList[i], param.Guid, v.SourceValue, v.Regulative, v.TargetValue, nowTime}})
	}
	return actions
}

func getLogMetricExprByAggType(metric, aggType, serviceGroup string, tagList []string) (result string) {
	switch aggType {
	case "sum":
		result = fmt.Sprintf("sum(%s{key=\"%s\",agg=\"%s\",service_group=\"%s\"}) by (key,agg,service_group)", models.LogMetricName, metric, aggType, serviceGroup)
	case "count":
		result = fmt.Sprintf("sum(%s{key=\"%s\",agg=\"%s\",service_group=\"%s\"}) by (key,agg,service_group)", models.LogMetricName, metric, aggType, serviceGroup)
	case "max":
		result = fmt.Sprintf("max(%s{key=\"%s\",agg=\"%s\",service_group=\"%s\"}) by (key,agg,service_group)", models.LogMetricName, metric, aggType, serviceGroup)
	case "min":
		result = fmt.Sprintf("min(%s{key=\"%s\",agg=\"%s\",service_group=\"%s\"}) by (key,agg,service_group)", models.LogMetricName, metric, aggType, serviceGroup)
	case "avg":
		result = fmt.Sprintf("sum(%s{key=\"%s\",agg=\"sum\",service_group=\"%s\"}) by (key,service_group)/sum(%s{key=\"%s\",agg=\"count\",service_group=\"%s\"}) by (key,service_group) > 0 or (0*sum(%s{key=\"%s\",agg=\"sum\",service_group=\"%s\"}) by (key,service_group))", models.LogMetricName, metric, serviceGroup, models.LogMetricName, metric, serviceGroup, models.LogMetricName, metric, serviceGroup)
	default:
		result = fmt.Sprintf("%s{key=\"%s\",agg=\"%s\",service_group=\"%s\"}", models.LogMetricName, metric, aggType, serviceGroup)
	}
	return result
}

func getLogMetricServiceGroup(logMetricMonitor string) (serviceGroup, monitorType string) {
	var logMetricMonitorTable []*models.LogMetricMonitorTable
	x.SQL("select guid,service_group,monitor_type from log_metric_monitor where guid=?", logMetricMonitor).Find(&logMetricMonitorTable)
	if len(logMetricMonitorTable) > 0 {
		serviceGroup = logMetricMonitorTable[0].ServiceGroup
		monitorType = logMetricMonitorTable[0].MonitorType
	}
	return
}

func getUpdateLogMetricConfigAction(param *models.LogMetricConfigObj, nowTime string) (actions []*Action, affectEndpointGroup []string) {
	param.Step = 10
	var logMetricConfigTable []*models.LogMetricConfigTable
	x.SQL("select * from log_metric_config where guid=?", param.Guid).Find(&logMetricConfigTable)
	if len(logMetricConfigTable) > 0 {
		if logMetricConfigTable[0].Metric != param.Metric || logMetricConfigTable[0].AggType != param.AggType {
			serviceGroup, _ := getLogMetricServiceGroup(param.LogMetricMonitor)
			oldMetricGuid := fmt.Sprintf("%s__%s", logMetricConfigTable[0].Metric, serviceGroup)
			newMetricGuid := fmt.Sprintf("%s__%s", param.Metric, serviceGroup)
			actions = append(actions, &Action{Sql: "update metric set guid=?,metric=?,prom_expr=? where guid=?", Param: []interface{}{newMetricGuid, param.Metric, getLogMetricExprByAggType(param.Metric, param.AggType, serviceGroup, []string{}), oldMetricGuid}})
			var alarmStrategyTable []*models.AlarmStrategyTable
			x.SQL("select guid,endpoint_group from alarm_strategy where metric=?", oldMetricGuid).Find(&alarmStrategyTable)
			if len(alarmStrategyTable) > 0 {
				for _, v := range alarmStrategyTable {
					affectEndpointGroup = append(affectEndpointGroup, v.EndpointGroup)
				}
				actions = append(actions, &Action{Sql: "update alarm_strategy set metric=? where metric=?", Param: []interface{}{newMetricGuid, oldMetricGuid}})
			}
		}
	}
	tagString := ""
	for _, jsonTagItem := range param.JsonTagList {
		param.TagConfig = append(param.TagConfig, &models.LogMetricConfigTag{Key: jsonTagItem})
	}
	if len(param.TagConfig) > 0 {
		tagBytes, _ := json.Marshal(param.TagConfig)
		tagString = string(tagBytes)
	}
	actions = append(actions, &Action{Sql: "update log_metric_config set metric=?,display_name=?,json_key=?,regular=?,agg_type=?,step=?,update_time=?,tag_config=? where guid=?", Param: []interface{}{param.Metric, param.DisplayName, param.JsonKey, param.Regular, param.AggType, param.Step, nowTime, tagString, param.Guid}})
	actions = append(actions, &Action{Sql: "delete from log_metric_string_map where log_metric_config=?", Param: []interface{}{param.Guid}})
	guidList := guid.CreateGuidList(len(param.StringMap))
	for i, v := range param.StringMap {
		actions = append(actions, &Action{Sql: "insert into log_metric_string_map(guid,log_metric_config,source_value,regulative,target_value,update_time) value (?,?,?,?,?,?)", Param: []interface{}{guidList[i], param.Guid, v.SourceValue, v.Regulative, v.TargetValue, nowTime}})
	}
	return
}

func getDeleteLogMetricConfigAction(logMetricConfigGuid, logMetricMonitorGuid string) (actions []*Action, endpointGroup []string) {
	lmObj, err := getSimpleLogMetricConfig(logMetricConfigGuid)
	if err != nil {
		log.Logger.Error("getDeleteLogMetricConfigAction", log.Error(err))
		return
	}
	serviceGroup, _ := getLogMetricServiceGroup(logMetricMonitorGuid)
	alarmMetricGuid := fmt.Sprintf("%s__%s", lmObj.Metric, serviceGroup)
	var alarmStrategyTable []*models.AlarmStrategyTable
	x.SQL("select guid,endpoint_group from alarm_strategy where metric=?", alarmMetricGuid).Find(&alarmStrategyTable)
	for _, v := range alarmStrategyTable {
		endpointGroup = append(endpointGroup, v.EndpointGroup)
	}
	actions = append(actions, &Action{Sql: "delete from alarm_strategy where metric=?", Param: []interface{}{alarmMetricGuid}})
	actions = append(actions, &Action{Sql: "delete from metric where guid=?", Param: []interface{}{alarmMetricGuid}})
	actions = append(actions, &Action{Sql: "delete from log_metric_string_map where log_metric_config=?", Param: []interface{}{logMetricConfigGuid}})
	actions = append(actions, &Action{Sql: "delete from log_metric_config where guid=?", Param: []interface{}{logMetricConfigGuid}})
	return
}

func getSimpleLogMetricConfig(logMetricConfigGuid string) (result models.LogMetricConfigTable, err error) {
	var queryTable []*models.LogMetricConfigTable
	err = x.SQL("select * from log_metric_config where guid=?", logMetricConfigGuid).Find(&queryTable)
	if err != nil {
		return result, err
	}
	if len(queryTable) == 0 {
		return result, fmt.Errorf("Can not find logMetricConfig with guid:%s ", logMetricConfigGuid)
	}
	result = *queryTable[0]
	return
}

func GetServiceGroupByLogMetricMonitor(logMetricMonitorGuid string) string {
	if logMetricMonitorGuid == "" {
		return ""
	}
	var logMetricMonitorTable []*models.LogMetricMonitorTable
	x.SQL("select * from log_metric_monitor where guid=?", logMetricMonitorGuid).Find(&logMetricMonitorTable)
	if len(logMetricMonitorTable) > 0 {
		return logMetricMonitorTable[0].ServiceGroup
	}
	return ""
}

func CheckRegExpMatchPCRE(param models.CheckRegExpParam) (message, matchString string) {
	re, tmpErr := pcre.Compile(param.RegString, 0)
	if tmpErr != nil {
		return fmt.Sprintf("reg compile fail,%s ", tmpErr.Message), matchString
	}
	matchString = pcreMatchSubString(&re, param.TestContext)
	if matchString == "" {
		return fmt.Sprintf("can not match any data"), matchString
	}
	return fmt.Sprintf("success match:%s", matchString), matchString
}

func CheckRegExpMatch(param models.CheckRegExpParam) (message string) {
	re, tmpErr := regexp2.Compile(param.RegString, 0)
	if tmpErr != nil {
		return fmt.Sprintf("reg compile fail,%s ", tmpErr.Error())
	}
	matchString := regexp2FindStringMatch(re, param.TestContext)
	if matchString == "" {
		return fmt.Sprintf("can not match any data")
	}
	return fmt.Sprintf("success match:%s", matchString)
}

func pcreMatchSubString(re *pcre.Regexp, lineText string) (matchString string) {
	if re == nil {
		return
	}
	mat := re.MatcherString(lineText, 0)
	if mat != nil {
		for i := 0; i <= mat.Groups(); i++ {
			groupString := mat.GroupString(i)
			if (i == 0 && groupString == lineText) || groupString == "" {
				continue
			}
			matchString = groupString
			break
		}
	}
	return
}

func regexp2FindStringMatch(re *regexp2.Regexp, lineText string) (matchString string) {
	if re == nil {
		return
	}
	mat, err := re.FindStringMatch(lineText)
	if err != nil || mat == nil {
		return
	}
	for i, v := range mat.Groups() {
		groupString := v.String()
		if (i == 0 && groupString == lineText) || groupString == "" {
			continue
		}
		matchString = groupString
		break
	}
	return
}

func ImportLogMetric(param *models.LogMetricQueryObj) (err error) {
	var actions []*Action
	existData, queryErr := GetLogMetricByServiceGroup(param.Guid)
	if queryErr != nil {
		return fmt.Errorf("get exist log metric data fail,%s ", queryErr.Error())
	}
	nowTime := time.Now().Format(models.DatetimeFormat)
	affectHostMap := make(map[string]int)
	affectEndpointGroupMap := make(map[string]int)
	//logMonitorMap := make(map[string]int)
	for _, inputLogMonitor := range param.Config {
		existObj := &models.LogMetricMonitorObj{}
		for _, existLogMonitor := range existData.Config {
			if existLogMonitor.Guid == inputLogMonitor.Guid {
				existObj = existLogMonitor
				break
			}
		}
		// log monitor action
		if existObj.Guid != "" {
			if existObj.LogPath != inputLogMonitor.LogPath || existObj.MonitorType != inputLogMonitor.MonitorType {
				actions = append(actions, &Action{Sql: "update log_metric_monitor set log_path=?,monitor_type=? where guid=?", Param: []interface{}{inputLogMonitor.LogPath, inputLogMonitor.MonitorType, inputLogMonitor.Guid}})
			}
		} else {
			actions = append(actions, &Action{Sql: "insert into log_metric_monitor(guid,service_group,log_path,metric_type,monitor_type,update_time) value (?,?,?,?,?,?)", Param: []interface{}{inputLogMonitor.Guid, param.Guid, inputLogMonitor.LogPath, inputLogMonitor.MetricType, inputLogMonitor.MonitorType, nowTime}})
		}
		tmpActions, tmpAffectHosts, tmpAffectEndpointGroup := getUpdateLogMetricMonitorByImport(existObj, inputLogMonitor, nowTime)
		actions = append(actions, tmpActions...)
		for _, v := range tmpAffectHosts {
			affectHostMap[v] = 1
		}
		for _, v := range tmpAffectEndpointGroup {
			affectEndpointGroupMap[v] = 1
		}
	}
	// delete action
	for _, existLogMonitor := range existData.Config {
		for _, v := range existLogMonitor.EndpointRel {
			affectHostMap[v.SourceEndpoint] = 1
		}
		deleteFlag := true
		for _, inputLogMonitor := range param.Config {
			if existLogMonitor.Guid == inputLogMonitor.Guid {
				deleteFlag = false
				break
			}
		}
		if deleteFlag {
			tmpDeleteActions, affectHost, affectEndpointGroup := getDeleteLogMetricMonitor(existLogMonitor.Guid)
			actions = append(actions, tmpDeleteActions...)
			for _, v := range affectHost {
				affectHostMap[v] = 1
			}
			for _, v := range affectEndpointGroup {
				affectEndpointGroupMap[v] = 1
			}
		}
	}
	var affectHostList, affectEndpointGroupList []string
	for k, _ := range affectHostMap {
		affectHostList = append(affectHostList, k)
	}
	for k, _ := range affectEndpointGroupMap {
		affectEndpointGroupList = append(affectEndpointGroupList, k)
	}
	log.Logger.Info("importActions", log.Int("length", len(actions)), log.StringList("affectHostList", affectHostList), log.StringList("affectEndpointGroupList", affectEndpointGroupList))
	err = Transaction(actions)
	if err != nil {
		log.Logger.Error("import log monitor exec database fail", log.Error(err))
		return
	}
	if tmpErr := SyncLogMetricExporterConfig(affectHostList); tmpErr != nil {
		log.Logger.Error("sync log metric to affect host fail", log.Error(tmpErr))
	}
	for _, v := range affectEndpointGroupList {
		if tmpErr := SyncPrometheusRuleFile(v, false); tmpErr != nil {
			log.Logger.Error("sync prometheus rule file fail", log.Error(tmpErr))
		}
	}
	return
}

func getUpdateLogMetricMonitorByImport(existObj, inputObj *models.LogMetricMonitorObj, nowTime string) (actions []*Action, affectHost []string, affectEndpointGroup []string) {
	if existObj.Guid != "" {
		// compare log json monitor
		for _, inputJsonObj := range inputObj.JsonConfigList {
			matchExistJsonObj := &models.LogMetricJsonObj{}
			for _, existJsonObj := range existObj.JsonConfigList {
				if existJsonObj.Guid == inputJsonObj.Guid {
					matchExistJsonObj = existJsonObj
					break
				}
			}
			if matchExistJsonObj.Guid != "" {
				actions = append(actions, &Action{Sql: "update log_metric_json set json_regular=?,tags=?,update_time=? where guid=?", Param: []interface{}{inputJsonObj.JsonRegular, inputJsonObj.Tags, nowTime, inputJsonObj.Guid}})
				tmpActions, tmpAffectEndpointGroup := getCompareLogMetricConfigByImport(inputJsonObj.MetricList, matchExistJsonObj.MetricList, nowTime)
				actions = append(actions, tmpActions...)
				affectEndpointGroup = append(affectEndpointGroup, tmpAffectEndpointGroup...)
			} else {
				actions = append(actions, &Action{Sql: "insert into log_metric_json(guid,log_metric_monitor,json_regular,tags,update_time) value (?,?,?,?,?)", Param: []interface{}{inputJsonObj.Guid, inputJsonObj.LogMetricMonitor, inputJsonObj.JsonRegular, inputJsonObj.Tags, nowTime}})
				for _, logMetricConfig := range inputJsonObj.MetricList {
					tmpActions := getCreateLogMetricConfigAction(logMetricConfig, nowTime)
					actions = append(actions, tmpActions...)
				}
			}
		}
		for _, existJsonObj := range existObj.JsonConfigList {
			deleteFlag := true
			for _, inputJsonObj := range inputObj.JsonConfigList {
				if inputJsonObj.Guid == existJsonObj.Guid {
					deleteFlag = false
					break
				}
			}
			if deleteFlag {
				for _, v := range existJsonObj.MetricList {
					deleteActions, tmpEndpointGroup := getDeleteLogMetricConfigByImport(v)
					actions = append(actions, deleteActions...)
					affectEndpointGroup = append(affectEndpointGroup, tmpEndpointGroup...)
				}
				actions = append(actions, &Action{Sql: "delete from log_metric_json where guid=?", Param: []interface{}{existJsonObj.Guid}})
			}
		}
		// compare log metric config
		tmpActions, tmpAffectEndpointGroup := getCompareLogMetricConfigByImport(inputObj.MetricConfigList, existObj.MetricConfigList, nowTime)
		actions = append(actions, tmpActions...)
		affectEndpointGroup = append(affectEndpointGroup, tmpAffectEndpointGroup...)
	} else {
		// create
		for _, inputJsonObj := range inputObj.JsonConfigList {
			actions = append(actions, &Action{Sql: "insert into log_metric_json(guid,log_metric_monitor,json_regular,tags,update_time) value (?,?,?,?,?)", Param: []interface{}{inputJsonObj.Guid, inputJsonObj.LogMetricMonitor, inputJsonObj.JsonRegular, inputJsonObj.Tags, nowTime}})
			for _, logMetricConfig := range inputJsonObj.MetricList {
				tmpActions := getCreateLogMetricConfigAction(logMetricConfig, nowTime)
				actions = append(actions, tmpActions...)
			}
		}
		for _, logMetricConfig := range inputObj.MetricConfigList {
			tmpActions := getCreateLogMetricConfigAction(logMetricConfig, nowTime)
			actions = append(actions, tmpActions...)
		}
	}
	return
}

func getCompareLogMetricConfigByImport(inputLogMetricList, existLogMetricList []*models.LogMetricConfigObj, nowTime string) (actions []*Action, affectEndpointGroup []string) {
	for _, inputLogMetricObj := range inputLogMetricList {
		matchExistMetricObj := &models.LogMetricConfigObj{}
		for _, existLogMetricObj := range existLogMetricList {
			if inputLogMetricObj.Guid == existLogMetricObj.Guid {
				matchExistMetricObj = existLogMetricObj
				break
			}
		}
		if matchExistMetricObj.Guid != "" {
			tmpActions, tmpAffectEndpointGroup := getUpdateLogMetricConfigByImport(inputLogMetricObj, matchExistMetricObj, nowTime)
			actions = append(actions, tmpActions...)
			affectEndpointGroup = append(affectEndpointGroup, tmpAffectEndpointGroup...)
		} else {
			tmpActions := getCreateLogMetricConfigAction(inputLogMetricObj, nowTime)
			actions = append(actions, tmpActions...)
		}
	}
	for _, existLogMetricObj := range existLogMetricList {
		deleteFlag := true
		for _, inputLogMetricObj := range inputLogMetricList {
			if inputLogMetricObj.Guid == existLogMetricObj.Guid {
				deleteFlag = false
				break
			}
		}
		if deleteFlag {
			deleteActions, tmpEndpointGroup := getDeleteLogMetricConfigByImport(existLogMetricObj)
			actions = append(actions, deleteActions...)
			affectEndpointGroup = append(affectEndpointGroup, tmpEndpointGroup...)
		}
	}
	return
}

func getUpdateLogMetricConfigByImport(inputLogMetric, existLogMetric *models.LogMetricConfigObj, nowTime string) (actions []*Action, affectEndpointGroup []string) {
	if existLogMetric.Metric != inputLogMetric.Metric || existLogMetric.AggType != inputLogMetric.AggType {
		oldMetricGuid := fmt.Sprintf("%s__%s", existLogMetric.Metric, inputLogMetric.ServiceGroup)
		newMetricGuid := fmt.Sprintf("%s__%s", inputLogMetric.Metric, inputLogMetric.ServiceGroup)
		actions = append(actions, &Action{Sql: "update metric set guid=?,metric=?,prom_expr=? where guid=?", Param: []interface{}{newMetricGuid, inputLogMetric.Metric, getLogMetricExprByAggType(inputLogMetric.Metric, inputLogMetric.AggType, inputLogMetric.ServiceGroup, []string{}), oldMetricGuid}})
		var alarmStrategyTable []*models.AlarmStrategyTable
		x.SQL("select guid,endpoint_group from alarm_strategy where metric=?", oldMetricGuid).Find(&alarmStrategyTable)
		if len(alarmStrategyTable) > 0 {
			for _, v := range alarmStrategyTable {
				affectEndpointGroup = append(affectEndpointGroup, v.EndpointGroup)
			}
			actions = append(actions, &Action{Sql: "update alarm_strategy set metric=? where metric=?", Param: []interface{}{newMetricGuid, oldMetricGuid}})
		}
	}
	actions = append(actions, &Action{Sql: "update log_metric_config set metric=?,display_name=?,json_key=?,regular=?,agg_type=?,step=?,update_time=? where guid=?", Param: []interface{}{inputLogMetric.Metric, inputLogMetric.DisplayName, inputLogMetric.JsonKey, inputLogMetric.Regular, inputLogMetric.AggType, inputLogMetric.Step, nowTime, inputLogMetric.Guid}})
	actions = append(actions, &Action{Sql: "delete from log_metric_string_map where log_metric_config=?", Param: []interface{}{inputLogMetric.Guid}})
	guidList := guid.CreateGuidList(len(inputLogMetric.StringMap))
	for i, v := range inputLogMetric.StringMap {
		actions = append(actions, &Action{Sql: "insert into log_metric_string_map(guid,log_metric_config,source_value,regulative,target_value,update_time) value (?,?,?,?,?,?)", Param: []interface{}{guidList[i], inputLogMetric.Guid, v.SourceValue, v.Regulative, v.TargetValue, nowTime}})
	}
	return
}

func getDeleteLogMetricConfigByImport(existLogMetric *models.LogMetricConfigObj) (actions []*Action, endpointGroup []string) {
	alarmMetricGuid := fmt.Sprintf("%s__%s", existLogMetric.Metric, existLogMetric.ServiceGroup)
	var alarmStrategyTable []*models.AlarmStrategyTable
	x.SQL("select guid,endpoint_group from alarm_strategy where metric=?", alarmMetricGuid).Find(&alarmStrategyTable)
	for _, v := range alarmStrategyTable {
		endpointGroup = append(endpointGroup, v.EndpointGroup)
	}
	actions = append(actions, &Action{Sql: "delete from alarm_strategy where metric=?", Param: []interface{}{alarmMetricGuid}})
	actions = append(actions, &Action{Sql: "delete from metric where guid=?", Param: []interface{}{alarmMetricGuid}})
	actions = append(actions, &Action{Sql: "delete from log_metric_string_map where log_metric_config=?", Param: []interface{}{existLogMetric.Guid}})
	actions = append(actions, &Action{Sql: "delete from log_metric_config where guid=?", Param: []interface{}{existLogMetric.Guid}})
	return
}

func ImportLogMetricExcel(logMonitorGuid string, param []*models.LogMetricConfigObj) (err error) {
	var actions []*Action
	var affectEndpointGroupList, affectHostList []string
	for _, v := range ListLogMetricEndpointRel(logMonitorGuid) {
		affectHostList = append(affectHostList, v.SourceEndpoint)
	}
	for _, existLogConfig := range ListLogMetricConfig("", logMonitorGuid) {
		tmpActions, tmpAffectEndpointGroup := getDeleteLogMetricConfigAction(existLogConfig.Guid, logMonitorGuid)
		actions = append(actions, tmpActions...)
		affectEndpointGroupList = append(affectEndpointGroupList, tmpAffectEndpointGroup...)
	}
	nowTime := time.Now().Format(models.DatetimeFormat)
	for _, inputLogConfig := range param {
		inputLogConfig.LogMetricMonitor = logMonitorGuid
		actions = append(actions, getCreateLogMetricConfigAction(inputLogConfig, nowTime)...)
	}
	err = Transaction(actions)
	if err != nil {
		log.Logger.Error("import log metric from excel exec database fail", log.Error(err))
		return
	}
	if tmpErr := SyncLogMetricExporterConfig(affectHostList); tmpErr != nil {
		log.Logger.Error("sync log metric to affect host fail", log.Error(tmpErr))
	}
	for _, v := range affectEndpointGroupList {
		if tmpErr := SyncPrometheusRuleFile(v, false); tmpErr != nil {
			log.Logger.Error("sync prometheus rule file fail", log.Error(tmpErr))
		}
	}
	return
}

func GetLogMetricByServiceGroupNew(serviceGroup string) (result models.LogMetricQueryObj, err error) {

	return
}

func GetSimpleLogMetricGroup(logMetricGroupGuid string) (result *models.LogMetricGroup, err error) {
	var logMetricGroupRows []*models.LogMetricGroup
	err = x.SQL("select * from log_metric_group where guid=?", logMetricGroupGuid).Find(&logMetricGroupRows)
	if err != nil {
		return result, fmt.Errorf("Query table log_metric_group fail,%s ", err.Error())
	}
	if len(logMetricGroupRows) == 0 {
		return result, fmt.Errorf("Can not find log_metric_group with guid:%s ", logMetricGroupGuid)
	}
	result = logMetricGroupRows[0]
	return
}

func GetLogMetricGroup(logMetricGroupGuid string) (result *models.LogMetricGroupWithTemplate, err error) {
	metricGroupObj, getGroupErr := GetSimpleLogMetricGroup(logMetricGroupGuid)
	if getGroupErr != nil {
		err = getGroupErr
		return
	}
	var logMetricStringMapRows []*models.LogMetricStringMapTable
	err = x.SQL("select * from log_metric_string_map where log_metric_group=?", logMetricGroupGuid).Find(&logMetricStringMapRows)
	if err != nil {
		return result, fmt.Errorf("Query table log_metric_string_map fail,%s ", err.Error())
	}
	result = &models.LogMetricGroupWithTemplate{LogMetricGroupGuid: logMetricGroupGuid, LogMetricMonitorGuid: metricGroupObj.LogMetricMonitor, LogMonitorTemplateGuid: metricGroupObj.LogMonitorTemplate, CodeStringMap: []*models.LogMetricStringMapTable{}, RetCodeStringMap: []*models.LogMetricStringMapTable{}}
	for _, row := range logMetricStringMapRows {
		if row.LogParamName == "code" {
			result.CodeStringMap = append(result.CodeStringMap, row)
		} else if row.LogParamName == "retcode" {
			result.RetCodeStringMap = append(result.RetCodeStringMap, row)
		}
	}
	return
}

func CreateLogMetricGroup(param *models.LogMetricGroupWithTemplate, operator string) (err error) {
	param.LogMetricGroupGuid = "lmg_" + guid.CreateGuid()
	logMonitorTemplateObj, getErr := GetLogMonitorTemplate(param.LogMonitorTemplateGuid)
	if getErr != nil {
		err = getErr
		return
	}
	nowTime := time.Now()
	var actions []*Action
	actions = append(actions, &Action{Sql: "insert into log_metric_group(guid,name,log_type,log_metric_monitor,log_monitor_template,create_user,create_time,update_user,update_time) values (?,?,?,?,?,?,?,?,?)", Param: []interface{}{
		param.LogMetricGroupGuid, logMonitorTemplateObj.Name, logMonitorTemplateObj.LogType, param.LogMetricMonitorGuid, param.LogMonitorTemplateGuid, operator, nowTime, operator, nowTime,
	}})
	actions = append(actions, getCreateLogMetricGroupMapAction(param, nowTime)...)
	// 自动添加增加 metric
	serviceGroup, monitorType := getLogMetricServiceGroup(param.LogMetricMonitorGuid)
	for _, v := range logMonitorTemplateObj.MetricList {
		actions = append(actions, &Action{Sql: "insert into metric(guid,metric,monitor_type,prom_expr,service_group,workspace,update_time) value (?,?,?,?,?,?,?)", Param: []interface{}{fmt.Sprintf("%s__%s", v.Metric, serviceGroup), v.Metric, monitorType, getLogMetricExprByAggType(v.Metric, v.AggType, serviceGroup, v.TagConfigList), serviceGroup, models.MetricWorkspaceService, nowTime}})
	}
	err = Transaction(actions)
	return
}

func getCreateLogMetricGroupMapAction(param *models.LogMetricGroupWithTemplate, nowTime time.Time) (actions []*Action) {
	codeGuidList := guid.CreateGuidList(len(param.CodeStringMap))
	for i, v := range param.CodeStringMap {
		actions = append(actions, &Action{Sql: "insert into log_metric_string_map(guid,log_metric_group,log_param_name,value_type,source_value,regulative,target_value,update_time) values (?,?,?,?,?,?,?,?)", Param: []interface{}{
			"lmsm_" + codeGuidList[i], param.LogMetricGroupGuid, "code", v.ValueType, v.SourceValue, v.Regulative, v.TargetValue, nowTime.Format(models.DatetimeFormat),
		}})
	}
	retCodeGuidList := guid.CreateGuidList(len(param.RetCodeStringMap))
	for i, v := range param.RetCodeStringMap {
		actions = append(actions, &Action{Sql: "insert into log_metric_string_map(guid,log_metric_group,log_param_name,value_type,source_value,regulative,target_value,update_time) values (?,?,?,?,?,?,?,?)", Param: []interface{}{
			"lmsm_" + retCodeGuidList[i], param.LogMetricGroupGuid, "retcode", v.ValueType, v.SourceValue, v.Regulative, v.TargetValue, nowTime.Format(models.DatetimeFormat),
		}})
	}
	return
}

func UpdateLogMetricGroup(param *models.LogMetricGroupWithTemplate, operator string) (err error) {
	nowTime := time.Now()
	var actions []*Action
	actions = append(actions, &Action{Sql: "update log_metric_group set update_user=?,update_time=? where guid=?", Param: []interface{}{
		operator, nowTime, param.LogMetricGroupGuid,
	}})
	actions = append(actions, &Action{Sql: "delete from log_metric_string_map where log_metric_group=?", Param: []interface{}{param.LogMetricGroupGuid}})
	actions = append(actions, getCreateLogMetricGroupMapAction(param, nowTime)...)
	err = Transaction(actions)
	return err
}

func DeleteLogMetricGroup(logMetricGroupGuid string) (logMetricMonitorGuid string, err error) {
	metricGroupObj, getGroupErr := GetSimpleLogMetricGroup(logMetricGroupGuid)
	if getGroupErr != nil {
		err = getGroupErr
		return
	}
	logMetricMonitorGuid = metricGroupObj.LogMetricMonitor
	var actions []*Action
	actions = append(actions, &Action{Sql: "delete from log_metric_string_map where log_metric_group=?", Param: []interface{}{logMetricGroupGuid}})
	actions = append(actions, &Action{Sql: "delete from log_metric_param where log_metric_group=?", Param: []interface{}{logMetricGroupGuid}})
	actions = append(actions, &Action{Sql: "delete from log_metric_config where log_metric_group=?", Param: []interface{}{logMetricGroupGuid}})
	actions = append(actions, &Action{Sql: "delete from log_metric_group where guid=?", Param: []interface{}{logMetricGroupGuid}})
	err = Transaction(actions)
	return
}

func ListLogMetricGroups(logMetricMonitor string) (result []*models.LogMetricGroupObj) {
	result = []*models.LogMetricGroupObj{}
	var logMetricGroupTable []*models.LogMetricGroup
	x.SQL("select * from log_metric_group where log_metric_monitor=?", logMetricMonitor).Find(&logMetricGroupTable)
	var templateGuidList []string
	for _, v := range logMetricGroupTable {
		if v.LogMonitorTemplate != "" {
			templateGuidList = append(templateGuidList, v.LogMonitorTemplate)
		}
		v.CreateTimeString = v.CreateTime.Format(models.DatetimeFormat)
		v.UpdateTimeString = v.UpdateTime.Format(models.DatetimeFormat)
		result = append(result, &models.LogMetricGroupObj{LogMetricGroup: *v})
	}
	templateNameMap := make(map[string]string)
	if len(templateGuidList) > 0 {
		var logMonitorTemplateRows []*models.LogMonitorTemplate
		filterSql, filterParam := createListParams(templateGuidList, "")
		x.SQL("select guid,name from log_monitor_template where guid in ("+filterSql+")", filterParam...).Find(&logMonitorTemplateRows)
		for _, row := range logMonitorTemplateRows {
			templateNameMap[row.Guid] = row.Name
		}
	}
	log.Logger.Debug("ListLogMetricGroups", log.JsonObj("templateNameMap", templateNameMap))
	for _, v := range result {
		if templateName, ok := templateNameMap[v.LogMonitorTemplate]; ok {
			log.Logger.Debug("ListLogMetricGroups match template name", log.JsonObj("templateName", templateName))
			v.LogMonitorTemplateName = templateName
		}
	}
	return result
}

func GetLogMetricCustomGroup(logMetricGroupGuid string) (result *models.LogMetricGroupObj, err error) {
	metricGroupObj, getGroupErr := GetSimpleLogMetricGroup(logMetricGroupGuid)
	if getGroupErr != nil {
		err = getGroupErr
		return
	}
	result = &models.LogMetricGroupObj{LogMetricGroup: *metricGroupObj, ParamList: []*models.LogMetricParamObj{}, MetricList: []*models.LogMetricConfigTable{}}
	var logMetricStringMapRows []*models.LogMetricStringMapTable
	err = x.SQL("select * from log_metric_string_map where log_metric_group=?", logMetricGroupGuid).Find(&logMetricStringMapRows)
	if err != nil {
		return result, fmt.Errorf("Query table log_metric_string_map fail,%s ", err.Error())
	}
	var logMetricParamRows []*models.LogMetricParam
	err = x.SQL("select * from log_metric_param where log_metric_group=?", logMetricGroupGuid).Find(&logMetricParamRows)
	if err != nil {
		return result, fmt.Errorf("Query table log_metric_param fail,%s ", err.Error())
	}
	for _, row := range logMetricParamRows {
		tmpParamObj := models.LogMetricParamObj{LogMetricParam: *row, StringMap: []*models.LogMetricStringMapTable{}}
		for _, stringMapRow := range logMetricStringMapRows {
			if stringMapRow.LogParamName == row.Name {
				tmpParamObj.StringMap = append(tmpParamObj.StringMap, stringMapRow)
			}
		}
		result.ParamList = append(result.ParamList, &tmpParamObj)
	}
	var logMetricConfigRows []*models.LogMetricConfigTable
	err = x.SQL("select * from log_metric_config where log_metric_group=?", logMetricGroupGuid).Find(&logMetricConfigRows)
	if err != nil {
		return result, fmt.Errorf("Query table log_metric_param fail,%s ", err.Error())
	}
	for _, row := range logMetricConfigRows {
		result.MetricList = append(result.MetricList, row)
	}
	return
}

func CreateLogMetricCustomGroup(param *models.LogMetricGroupObj, operator string) (err error) {
	param.LogType = "custom"
	param.Guid = "lmg_" + guid.CreateGuid()
	nowTime := time.Now()
	var actions []*Action
	actions = append(actions, &Action{Sql: "insert into log_metric_group(guid,name,log_type,log_metric_monitor,demo_log,calc_result,create_user,create_time,update_user,update_time) values (?,?,?,?,?,?,?,?,?,?)", Param: []interface{}{
		param.Guid, param.Name, param.LogType, param.LogMetricMonitor, param.DemoLog, param.CalcResult, operator, nowTime, operator, nowTime,
	}})
	paramGuidList := guid.CreateGuidList(len(param.ParamList))
	for i, v := range param.ParamList {
		actions = append(actions, &Action{Sql: "insert into log_metric_param(guid,name,display_name,log_metric_group,regular,demo_match_value,create_user,create_time) values (?,?,?,?,?,?,?,?)", Param: []interface{}{
			"lmp_" + paramGuidList[i], v.Name, v.DisplayName, param.Guid, v.Regular, v.DemoMatchValue, operator, nowTime,
		}})
	}
	metricGuidList := guid.CreateGuidList(len(param.MetricList))
	for i, v := range param.MetricList {
		tmpTagListBytes, _ := json.Marshal(v.TagConfigList)
		actions = append(actions, &Action{Sql: "insert into log_metric_config(guid,log_metric_monitor,log_metric_group,log_param_name,metric,display_name,regular,step,agg_type,tag_config,create_user,create_time) values (?,?,?,?,?,?,?,?,?,?,?,?)", Param: []interface{}{
			"lmc_" + metricGuidList[i], param.LogMetricMonitor, param.Guid, v.LogParamName, v.Metric, v.DisplayName, v.Regular, v.Step, v.AggType, string(tmpTagListBytes), operator, nowTime,
		}})
	}
	err = Transaction(actions)
	return
}

func UpdateLogMetricCustomGroup(param *models.LogMetricGroupObj, operator string) (err error) {
	existLogGroupData, getExistErr := GetLogMetricCustomGroup(param.Guid)
	if getExistErr != nil {
		err = getExistErr
		return
	}
	nowTime := time.Now()
	var actions []*Action
	actions = append(actions, &Action{Sql: "update log_metric_group set name=?,demo_log=?,calc_result=?,update_user=?,update_time=? where guid=?", Param: []interface{}{
		param.Name, param.DemoLog, param.CalcResult, operator, nowTime, param.Guid,
	}})
	paramGuidList := guid.CreateGuidList(len(param.ParamList))
	for i, inputParamObj := range param.ParamList {
		if inputParamObj.Guid == "" {
			actions = append(actions, &Action{Sql: "insert into log_metric_param(guid,name,display_name,log_metric_group,regular,demo_match_value,create_user,create_time) values (?,?,?,?,?,?,?,?)", Param: []interface{}{
				"lmp_" + paramGuidList[i], inputParamObj.Name, inputParamObj.DisplayName, param.Guid, inputParamObj.Regular, inputParamObj.DemoMatchValue, operator, nowTime,
			}})
		} else {
			actions = append(actions, &Action{Sql: "update log_metric_param set name=?,display_name=?,regular=?,demo_match_value=?,update_user=?,update_time=? where guid=?", Param: []interface{}{
				inputParamObj.Name, inputParamObj.DisplayName, inputParamObj.Regular, inputParamObj.DemoMatchValue, operator, nowTime, inputParamObj.Guid,
			}})
		}
	}
	for _, existParamObj := range existLogGroupData.ParamList {
		deleteFlag := true
		for _, inputParamObj := range param.ParamList {
			if inputParamObj.Guid == existParamObj.Guid {
				deleteFlag = false
				break
			}
		}
		if deleteFlag {
			actions = append(actions, &Action{Sql: "delete from log_metric_param where guid=?", Param: []interface{}{existParamObj.Guid}})
		}
	}
	metricGuidList := guid.CreateGuidList(len(param.MetricList))
	for i, inputMetricObj := range param.MetricList {
		tmpTagListBytes, _ := json.Marshal(inputMetricObj.TagConfigList)
		if inputMetricObj.Guid == "" {
			actions = append(actions, &Action{Sql: "insert into log_metric_config(guid,log_metric_monitor,log_metric_group,log_param_name,metric,display_name,regular,step,agg_type,tag_config,create_user,create_time) values (?,?,?,?,?,?,?,?,?,?,?,?)", Param: []interface{}{
				"lmc_" + metricGuidList[i], existLogGroupData.LogMetricMonitor, param.Guid, inputMetricObj.LogParamName, inputMetricObj.Metric, inputMetricObj.DisplayName, inputMetricObj.Regular, inputMetricObj.Step, inputMetricObj.AggType, string(tmpTagListBytes), operator, nowTime,
			}})
		} else {
			actions = append(actions, &Action{Sql: "update log_metric_config set log_param_name=?,metric=?,display_name=?,regular=?,step=?,agg_type=?,tag_config=?,update_user=?,update_time=? where guid=?", Param: []interface{}{
				inputMetricObj.LogParamName, inputMetricObj.Metric, inputMetricObj.DisplayName, inputMetricObj.Regular, inputMetricObj.Step, inputMetricObj.AggType, string(tmpTagListBytes), operator, nowTime, inputMetricObj.Guid,
			}})
		}
	}
	for _, existMetricObj := range existLogGroupData.MetricList {
		deleteFlag := true
		for _, inputMetricObj := range param.MetricList {
			if inputMetricObj.Guid == existMetricObj.Guid {
				deleteFlag = false
				break
			}
		}
		if deleteFlag {
			actions = append(actions, &Action{Sql: "delete from log_metric_config where guid=?", Param: []interface{}{existMetricObj.Guid}})
		}
	}
	err = Transaction(actions)
	return err
}
