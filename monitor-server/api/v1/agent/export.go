package agent

import (
	"github.com/gin-gonic/gin"
	"strings"
	mid "github.com/WeBankPartners/wecube-plugins-prometheus/monitor-server/middleware"
	m "github.com/WeBankPartners/wecube-plugins-prometheus/monitor-server/models"
	"net/http"
	"fmt"
	"github.com/WeBankPartners/wecube-plugins-prometheus/monitor-server/services/db"
	"net/url"
	"io/ioutil"
	"encoding/json"
)

type resultObj struct {
	ResultCode  string  `json:"result_code"`
	ResultMessage  string  `json:"result_message"`
}

type requestObj struct {
	RequestId  string  	`json:"requestId"`
	Inputs  []hostRequestObj  `json:"inputs"`
}

type hostRequestObj struct {
	CallbackParameter  string  `json:"callbackParameter"`
	HostIp  string  `json:"host_ip"`
}

func StartHostAgentNew(c *gin.Context)  {
	data,_ := ioutil.ReadAll(c.Request.Body)
	mid.LogInfo(fmt.Sprintf("param : %v", string(data)))
	var param requestObj
	var result resultObj
	err := json.Unmarshal(data, &param)
	if err == nil {
		if len(param.Inputs) == 0 {
			result = resultObj{ResultCode:"1", ResultMessage:"Param validate fail : inputs length is zero"}
			mid.LogInfo(fmt.Sprintf("result : code %s , message %s", result.ResultCode, result.ResultMessage))
			c.JSON(http.StatusBadRequest, result)
			return
		}
		var ipList []string
		var tmpHostIp string
		if strings.Contains(param.Inputs[0].HostIp, "[") || strings.Contains(param.Inputs[0].HostIp, ",") {
			tmpHostIp = strings.ReplaceAll(param.Inputs[0].HostIp, "[", "")
			tmpHostIp = strings.ReplaceAll(param.Inputs[0].HostIp, "]", "")
			ipList = strings.Split(tmpHostIp, ",")
		}else{
			ipList = append(ipList, param.Inputs[0].HostIp)
		}
		for _,hostIp := range ipList {
			param := m.RegisterParam{Type:hostType, ExporterIp:hostIp, ExporterPort:"9100"}
			err := RegisterJob(param)
			if err != nil {
				result = resultObj{ResultCode:"1", ResultMessage:fmt.Sprintf("register %s:%s fail,error %v",hostType, hostIp, err)}
				mid.LogInfo(fmt.Sprintf("result : code %s , message %s", result.ResultCode, result.ResultMessage))
				c.JSON(http.StatusInternalServerError, result)
				return
			}
		}
		result = resultObj{ResultCode:"0", ResultMessage:"Success"}
		mid.LogInfo(fmt.Sprintf("result : code %s , message %s", result.ResultCode, result.ResultMessage))
		mid.ReturnData(c, result)
	}else{
		result = resultObj{ResultCode:"1", ResultMessage:fmt.Sprintf("Param validate fail : %v", err)}
		mid.LogInfo(fmt.Sprintf("result : code %s , message %s", result.ResultCode, result.ResultMessage))
		c.JSON(http.StatusBadRequest, result)
	}
}

func StartHostAgent(c *gin.Context)  {
	hostIp := c.Query("host_ip")
	//osType := strings.ToLower(c.Query("os_type"))
	//if !isLinuxType(osType) {
	//	mid.ReturnValidateFail(c, "Illegal OS type")
	//	return
	//}
	param := m.RegisterParam{Type:hostType, ExporterIp:hostIp, ExporterPort:"9100"}
	err := RegisterJob(param)
	if err != nil {
		c.JSON(http.StatusInternalServerError, resultObj{ResultCode:"1", ResultMessage:fmt.Sprintf("register %s:%s fail,error %v",hostType, hostIp, err)})
		return
	}
	mid.ReturnData(c, resultObj{ResultCode:"0", ResultMessage:"Success"})
}

func StopHostAgent(c *gin.Context)  {
	hostIp := c.Query("host_ip")
	endpointObj := m.EndpointTable{Ip:hostIp, ExportType:hostType}
	db.GetEndpoint(&endpointObj)
	if endpointObj.Id <= 0 {
		c.JSON(http.StatusInternalServerError, resultObj{ResultCode:"1", ResultMessage:fmt.Sprintf("deregister %s:%s fail,can't find this host",hostType, hostIp)})
		return
	}
	err := DeregisterJob(endpointObj.Guid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, resultObj{ResultCode:"1", ResultMessage:fmt.Sprintf("deregister %s:%s fail,error %v",hostType, hostIp, err)})
		return
	}
	mid.ReturnData(c, resultObj{ResultCode:"0", ResultMessage:"Success"})
}

func StartMysqlAgent(c *gin.Context)  {
	hostIp := c.Query("host_ip")
	instance := c.Query("instance")
	if instance == "" {
		mid.ReturnValidateFail(c, "Instance can not be empty")
		return
	}
	param := m.RegisterParam{Type:mysqlType, ExporterIp:hostIp, ExporterPort:"9104", Instance:instance}
	err := RegisterJob(param)
	if err != nil {
		c.JSON(http.StatusInternalServerError, resultObj{ResultCode:"1", ResultMessage:fmt.Sprintf("register %s:%s fail,error %v",mysqlType, hostIp, err)})
		return
	}
	mid.ReturnData(c, resultObj{ResultCode:"0", ResultMessage:"Success"})
}

func StopMysqlAgent(c *gin.Context)  {
	hostIp := c.Query("host_ip")
	instance := c.Query("instance")
	if instance == "" {
		mid.ReturnValidateFail(c, "Instance can not be empty")
		return
	}
	endpointObj := m.EndpointTable{Ip:hostIp, ExportType:mysqlType, Name:instance}
	db.GetEndpoint(&endpointObj)
	if endpointObj.Id <= 0 {
		c.JSON(http.StatusInternalServerError, resultObj{ResultCode:"1", ResultMessage:fmt.Sprintf("deregister %s:%s fail,can't find this host",mysqlType, hostIp)})
		return
	}
	err := DeregisterJob(endpointObj.Guid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, resultObj{ResultCode:"1", ResultMessage:fmt.Sprintf("deregister %s:%s fail,error %v",mysqlType, hostIp, err)})
		return
	}
	mid.ReturnData(c, resultObj{ResultCode:"0", ResultMessage:"Success"})
}

func StartRedisAgent(c *gin.Context)  {
	hostIp := c.Query("host_ip")
	instance := c.Query("instance")
	if instance == "" {
		mid.ReturnValidateFail(c, "Instance can not be empty")
		return
	}
	param := m.RegisterParam{Type:redisType, ExporterIp:hostIp, ExporterPort:"9121", Instance:instance}
	err := RegisterJob(param)
	if err != nil {
		c.JSON(http.StatusInternalServerError, resultObj{ResultCode:"1", ResultMessage:fmt.Sprintf("register %s:%s fail,error %v",redisType, hostIp, err)})
		return
	}
	mid.ReturnData(c, resultObj{ResultCode:"0", ResultMessage:"Success"})
}

func StopRedisAgent(c *gin.Context)  {
	hostIp := c.Query("host_ip")
	instance := c.Query("instance")
	if instance == "" {
		mid.ReturnValidateFail(c, "Instance can not be empty")
		return
	}
	endpointObj := m.EndpointTable{Ip:hostIp, ExportType:redisType, Name:instance}
	db.GetEndpoint(&endpointObj)
	if endpointObj.Id <= 0 {
		c.JSON(http.StatusInternalServerError, resultObj{ResultCode:"1", ResultMessage:fmt.Sprintf("deregister %s:%s fail,can't find this host",redisType, hostIp)})
		return
	}
	err := DeregisterJob(endpointObj.Guid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, resultObj{ResultCode:"1", ResultMessage:fmt.Sprintf("deregister %s:%s fail,error %v",redisType, hostIp, err)})
		return
	}
	mid.ReturnData(c, resultObj{ResultCode:"0", ResultMessage:"Success"})
}

func StartTomcatAgent(c *gin.Context)  {
	hostIp := c.Query("host_ip")
	instance := c.Query("instance")
	if instance == "" {
		mid.ReturnValidateFail(c, "Instance can not be empty")
		return
	}
	param := m.RegisterParam{Type:tomcatType, ExporterIp:hostIp, ExporterPort:"9151", Instance:instance}
	err := RegisterJob(param)
	if err != nil {
		c.JSON(http.StatusInternalServerError, resultObj{ResultCode:"1", ResultMessage:fmt.Sprintf("register %s:%s fail,error %v",tomcatType, hostIp, err)})
		return
	}
	mid.ReturnData(c, resultObj{ResultCode:"0", ResultMessage:"Success"})
}

func StopTomcatAgent(c *gin.Context)  {
	hostIp := c.Query("host_ip")
	instance := c.Query("instance")
	if instance == "" {
		mid.ReturnValidateFail(c, "Instance can not be empty")
		return
	}
	endpointObj := m.EndpointTable{Ip:hostIp, ExportType:tomcatType, Name:instance}
	db.GetEndpoint(&endpointObj)
	if endpointObj.Id <= 0 {
		c.JSON(http.StatusInternalServerError, resultObj{ResultCode:"1", ResultMessage:fmt.Sprintf("deregister %s:%s fail,can't find this host",tomcatType, hostIp)})
		return
	}
	err := DeregisterJob(endpointObj.Guid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, resultObj{ResultCode:"1", ResultMessage:fmt.Sprintf("deregister %s:%s fail,error %v",tomcatType, hostIp, err)})
		return
	}
	mid.ReturnData(c, resultObj{ResultCode:"0", ResultMessage:"Success"})
}

func GetSystemDashboardUrl(c *gin.Context)  {
	name := c.Query("system_name")
	ips := c.Query("ips")
	urlParms := url.Values{}
	urlParms.Set("systemName", name)
	urlParms.Set("ips", ips)
	urlPath := fmt.Sprintf("http://%s/wecube-monitor/#/systemMonitoring?%s", c.Request.Host, urlParms.Encode())
	mid.LogInfo(fmt.Sprintf("url : %s", urlPath))
	mid.ReturnData(c, resultObj{ResultCode:"0", ResultMessage:urlPath})
}

func isLinuxType(osType string) bool {
	linuxType := []string{"linux", "redhat", "centos", "ubuntu", "unix"}
	result := false
	for _,v := range linuxType {
		if strings.Contains(osType, v) {
			result = true
			break
		}
	}
	return result
}
