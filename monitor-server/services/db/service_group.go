package db

import (
	"fmt"
	"github.com/WeBankPartners/open-monitor/monitor-server/middleware/log"
	"github.com/WeBankPartners/open-monitor/monitor-server/models"
	"strings"
	"sync"
)

var (
	globalServiceGroupMap  = make(map[string]*models.ServiceGroupLinkNode)
	globalServiceGroupLock = new(sync.RWMutex)
)

func InitServiceGroup() {
	var serviceGroupTable []*models.ServiceGroupTable
	err := x.SQL("select guid,parent,display_name,service_type from service_group").Find(&serviceGroupTable)
	if err != nil {
		log.Logger.Error("Init service group fail", log.Error(err))
		return
	}
	if len(serviceGroupTable) == 0 {
		return
	}
	buildGlobalServiceGroupLink(serviceGroupTable)
}

func buildGlobalServiceGroupLink(serviceGroupTable []*models.ServiceGroupTable) {
	globalServiceGroupLock.Lock()
	globalServiceGroupMap = make(map[string]*models.ServiceGroupLinkNode)
	for _, v := range serviceGroupTable {
		globalServiceGroupMap[v.Guid] = &models.ServiceGroupLinkNode{Guid: v.Guid}
	}
	for _, v := range serviceGroupTable {
		if v.Parent != "" {
			globalServiceGroupMap[v.Guid].Parent = globalServiceGroupMap[v.Parent]
			globalServiceGroupMap[v.Parent].Children = append(globalServiceGroupMap[v.Parent].Children, globalServiceGroupMap[v.Guid])
		}
	}
	globalServiceGroupLock.Unlock()
}

func ListServiceGroup() (result []*models.ServiceGroupTable, err error) {
	result = []*models.ServiceGroupTable{}
	err = x.SQL("select * from service_group").Find(&result)
	return
}

func GetServiceGroupEndpointList(searchType string) (result []*models.ServiceGroupEndpointListObj, err error) {
	result = []*models.ServiceGroupEndpointListObj{}
	if searchType == "endpoint" {
		var endpointTable []*models.EndpointNew
		err = x.SQL("select guid from endpoint_new").Find(&endpointTable)
		for _, v := range endpointTable {
			result = append(result, &models.ServiceGroupEndpointListObj{Guid: v.Guid, DisplayName: v.Guid})
		}
	} else {
		var serviceGroupTable []*models.ServiceGroupTable
		err = x.SQL("select guid,display_name from service_group").Find(&serviceGroupTable)
		for _, v := range serviceGroupTable {
			result = append(result, &models.ServiceGroupEndpointListObj{Guid: v.Guid, DisplayName: v.DisplayName})
		}
	}
	return
}

func CreateServiceGroup(param models.ServiceGroupTable) {
	globalServiceGroupLock.Lock()
	if param.Parent != "" {

	}
	globalServiceGroupLock.Unlock()
}

func UpdateServiceGroup() {

}

func DeleteServiceGroup() {

}

func ListServiceGroupEndpoint(serviceGroup, monitorType string) (result []*models.ServiceGroupEndpointListObj, err error) {
	if _, b := globalServiceGroupMap[serviceGroup]; !b {
		return result, fmt.Errorf("Can not find service_group:%s ", serviceGroup)
	}
	guidList := globalServiceGroupMap[serviceGroup].FetchChildGuid()
	result = []*models.ServiceGroupEndpointListObj{}
	var endpointServiceRel []*models.EndpointServiceRelTable
	err = x.SQL("select distinct t1.endpoint from endpoint_service_rel t1 left join endpoint_new t2 on t1.endpoint=t2.guid where t1.service_group in ('"+strings.Join(guidList, "','")+"') and t2.monitor_type=?", monitorType).Find(&endpointServiceRel)
	for _, v := range endpointServiceRel {
		result = append(result, &models.ServiceGroupEndpointListObj{Guid: v.Endpoint, DisplayName: v.Endpoint})
	}
	return
}

func getSimpleServiceGroup(serviceGroupGuid string) (result models.ServiceGroupTable, err error) {
	var serviceGroupTable []*models.ServiceGroupTable
	err = x.SQL("select * from service_group where guid=?", serviceGroupGuid).Find(&serviceGroupTable)
	if err != nil {
		return result, fmt.Errorf("Query service_group table fail,%s ", err.Error())
	}
	if len(serviceGroupTable) == 0 {
		return result, fmt.Errorf("Can not find service_group with guid:%s ", serviceGroupGuid)
	}
	result = *serviceGroupTable[0]
	return
}
