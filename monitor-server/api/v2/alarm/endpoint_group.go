package alarm

import (
	"github.com/WeBankPartners/open-monitor/monitor-server/middleware"
	"github.com/WeBankPartners/open-monitor/monitor-server/models"
	"github.com/WeBankPartners/open-monitor/monitor-server/services/db"
	"github.com/gin-gonic/gin"
)

func ListEndpointGroup(c *gin.Context) {
	var param models.QueryRequestParam
	if err := c.ShouldBindJSON(&param); err != nil {
		middleware.ReturnValidateError(c, err.Error())
		return
	}
	pageInfo, rowData, err := db.ListEndpointGroup(&param)
	if err != nil {
		middleware.ReturnHandleError(c, err.Error(), err)
	} else {
		middleware.ReturnPageData(c, pageInfo, rowData)
	}
}

func CreateEndpointGroup(c *gin.Context) {
	var param models.EndpointGroupTable
	if err := c.ShouldBindJSON(&param); err != nil {
		middleware.ReturnValidateError(c, err.Error())
		return
	}
	err := db.CreateEndpointGroup(&param)
	if err != nil {
		middleware.ReturnHandleError(c, err.Error(), err)
	} else {
		middleware.ReturnSuccess(c)
	}
}

func UpdateEndpointGroup(c *gin.Context) {
	var param models.EndpointGroupTable
	if err := c.ShouldBindJSON(&param); err != nil {
		middleware.ReturnValidateError(c, err.Error())
		return
	}
	err := db.UpdateEndpointGroup(&param)
	if err != nil {
		middleware.ReturnHandleError(c, err.Error(), err)
	} else {
		middleware.ReturnSuccess(c)
	}
}

func DeleteEndpointGroup(c *gin.Context) {
	endpointGroupGuid := c.Param("groupGuid")
	err := db.DeleteEndpointGroup(endpointGroupGuid)
	if err != nil {
		middleware.ReturnHandleError(c, err.Error(), err)
	} else {
		middleware.ReturnSuccess(c)
	}
}

func GetGroupEndpointRel(c *gin.Context) {
	endpointGroupGuid := c.Param("groupGuid")
	result, err := db.GetGroupEndpointRel(endpointGroupGuid)
	if err != nil {
		middleware.ReturnHandleError(c, err.Error(), err)
	} else {
		middleware.ReturnSuccessData(c, result)
	}
}

func UpdateGroupEndpoint(c *gin.Context) {
	var param models.UpdateGroupEndpointParam
	if err := c.ShouldBindJSON(&param); err != nil {
		middleware.ReturnValidateError(c, err.Error())
		return
	}
	err := db.UpdateGroupEndpoint(&param)
	if err != nil {
		middleware.ReturnHandleError(c, err.Error(), err)
	} else {
		middleware.ReturnSuccess(c)
	}
}

func GetGroupEndpointNotify(c *gin.Context) {
	endpointGroupGuid := c.Param("groupGuid")
	result, err := db.GetGroupEndpointNotify(endpointGroupGuid)
	if err != nil {
		middleware.ReturnHandleError(c, err.Error(), err)
	} else {
		middleware.ReturnSuccessData(c, result)
	}
}

func UpdateGroupEndpointNotify(c *gin.Context) {
	endpointGroupGuid := c.Param("groupGuid")
	var param []*models.NotifyObj
	if err := c.ShouldBindJSON(&param); err != nil {
		middleware.ReturnValidateError(c, err.Error())
		return
	}
	err := db.UpdateGroupEndpointNotify(endpointGroupGuid, param)
	if err != nil {
		middleware.ReturnHandleError(c, err.Error(), err)
	} else {
		middleware.ReturnSuccess(c)
	}
}
