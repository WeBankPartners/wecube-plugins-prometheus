package middleware

import (
	"bytes"
	"encoding/gob"
	"time"
	m "github.com/WeBankPartners/open-monitor/monitor-server/models"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sync"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"strings"
	"encoding/base64"
	"encoding/json"
	"github.com/WeBankPartners/open-monitor/monitor-server/middleware/log"
	"io/ioutil"
	"github.com/dgrijalva/jwt-go"
	"strconv"
)

var RedisClient *redis.Client
var LocalMem = make(map[string]m.Session)
var onlyLocalStore bool
var localStoreLock = new(sync.RWMutex)
var expireTime = int64(3600)
var RecordRequestMap = make(map[string]int64)

func InitSession()  {
	sessionConfig := m.Config().Http.Session
	expireTime = m.Config().Http.Session.Expire
	onlyLocalStore = true
	if sessionConfig.Redis.Enable {
		client := redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", sessionConfig.Redis.Server, sessionConfig.Redis.Port),
			Password: sessionConfig.Redis.Pwd, // no password set
			DB:       sessionConfig.Redis.Db,  // use default DB
		})
		_, err := client.Ping().Result()
		if err!=nil {
			log.Logger.Error("Init session redis fail", log.Error(err))
			onlyLocalStore = true
		}else{
			log.Logger.Info("init session redis success")
			onlyLocalStore = false
			RedisClient = client
		}
	}
}

func SaveSession(session m.Session) (isOk bool,sId string) {
	isOk = true
	session.Expire = time.Now().Unix() + expireTime
	serializeData,err := serialize(session)
	if err != nil {
		log.Logger.Error("Serialize session error", log.Error(err))
		return false, sId
	}
	md := md5.New()
	md.Write(serializeData)
	if session.Token != "" {
		sId = session.Token
	}else {
		sId = hex.EncodeToString(md.Sum(nil))
	}
	if !onlyLocalStore {
		backCmd := RedisClient.Set(fmt.Sprintf("session_%s", sId), serializeData, time.Duration(expireTime) * time.Second)
		if !strings.Contains(backCmd.Val(), "OK") {
			log.Logger.Error("Save session to redis fail", log.Error(err))
			return false, sId
		}
	}
	localStoreLock.Lock()
	LocalMem[sId] = session
	localStoreLock.Unlock()
	return isOk, sId
}

func GetOperateUser(c *gin.Context) string {
	if !m.Config().Http.Session.Enable {
		return ""
	}
	auToken := c.GetHeader("X-Auth-Token")
	if auToken!= "" {
		if m.Config().Http.Session.ServerEnable {
			if auToken == m.Config().Http.Session.ServerToken {
				return "auth_server"
			}
		}
		session := GetSessionData(auToken)
		return fmt.Sprintf("%s", session.User)
	}else{
		ReturnTokenError(c)
		return ""
	}
}

func GetCoreRequestRoleList(c *gin.Context) []string {
	var result []string
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return result
	}
	authHeader = authHeader[strings.Index(authHeader, ".")+1:]
	authHeader = authHeader[:strings.LastIndex(authHeader, ".")]
	authHeader += "=="
	b,err := base64.StdEncoding.DecodeString(authHeader)
	if err != nil {
		log.Logger.Error("Decode core request base64 fail", log.Error(err))
		return result
	}
	var requestToke m.CoreRequestToken
	err = json.Unmarshal(b, &requestToke)
	if err != nil {
		log.Logger.Error("Get core token,json unmarchal fail", log.Error(err))
		return result
	}
	if requestToke.Authority != "" {
		requestToke.Authority = strings.Replace(requestToke.Authority, "[", "", -1)
		requestToke.Authority = strings.Replace(requestToke.Authority, "]", "", -1)
		result = strings.Split(requestToke.Authority, ",")
	}
	return result
}

func GetSessionData(sId string) m.Session {
	var result m.Session
	localContain := false
	localStoreLock.RLock()
	if v,i := LocalMem[sId];i {
		result = v
		localContain = true
	}
	localStoreLock.RUnlock()
	if !localContain && !onlyLocalStore {
		re := RedisClient.Get(fmt.Sprintf("session_%s", sId))
		if len(re.Val()) > 0 {
			deserialize([]byte(re.Val()), &result)
			LocalMem[sId] = result
		}
	}
	return result
}

func IsActive(sId string, clientIp string) bool {
	if m.Config().Http.Session.ServerEnable {
		if sId == m.Config().Http.Session.ServerToken {
			return true
		}
	}
	var tmpUser string
	localContain := false
	//localStoreLock.RLock()
	//defer localStoreLock.RUnlock()
	if v,i := LocalMem[sId];i {
		tmpUser = v.User
		if time.Now().Unix() > v.Expire {
			if rrm,b := RecordRequestMap[fmt.Sprintf("%s_%s", tmpUser,clientIp)]; b{
				if time.Now().Unix()-rrm <= expireTime {
					localContain = true
					tmpSession := m.Session{User:v.User, Token:sId}
					SaveSession(tmpSession)
				}
			}
			if !localContain {
				delete(LocalMem, sId)
				return false
			}
		}
		localContain = true
	}
	if !localContain && !onlyLocalStore {
		var result m.Session
		re := RedisClient.Get(fmt.Sprintf("session_%s", sId))
		if len(re.Val()) > 0 {
			deserialize([]byte(re.Val()), &result)
			tmpUser = result.User
			LocalMem[sId] = result
			localContain = true
		}
	}
	if localContain {
		RecordRequestMap[fmt.Sprintf("%s_%s", tmpUser,clientIp)] = time.Now().Unix()
	}
	return localContain
}

func DelSession(sId string) {
	localStoreLock.Lock()
	if _,i := LocalMem[sId];i {
		delete(LocalMem, sId)
	}
	localStoreLock.Unlock()
	if !onlyLocalStore {
		RedisClient.Del(sId)
	}
}

// Serialize encodes a value using gob.
func serialize(src interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(src); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Deserialize decodes a value using gob.
func deserialize(src []byte, dst interface{}) error {
	dec := gob.NewDecoder(bytes.NewBuffer(src))
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return nil
}

func DecodeCoreToken(token,key string) (result m.CoreJwtToken,err error) {
	if strings.HasPrefix(token, "Bearer") {
		token = token[7:]
	}
	if key == "" || strings.HasPrefix(key, "{{") {
		key = "Platform+Auth+Server+Secret"
	}
	keyBytes,err := ioutil.ReadAll(base64.NewDecoder(base64.RawStdEncoding, bytes.NewBufferString(key)))
	if err != nil {
		log.Logger.Error("Decode core token fail,base64 decode error", log.Error(err))
		return result,err
	}
	pToken,err := jwt.Parse(token, func(*jwt.Token) (interface{}, error) {
		return keyBytes, nil
	})
	if err != nil {
		log.Logger.Error("Decode core token fail,jwt parse error", log.Error(err))
		return result,err
	}
	claimMap,ok := pToken.Claims.(jwt.MapClaims)
	if !ok {
		log.Logger.Error("Decode core token fail,claims to map error", log.Error(err))
		return result,err
	}
	result.User = fmt.Sprintf("%s", claimMap["sub"])
	result.Expire,err = strconv.ParseInt(fmt.Sprintf("%.0f", claimMap["exp"]), 10, 64)
	if err != nil {
		log.Logger.Error("Decode core token fail,parse expire to int64 error", log.Error(err))
		return result,err
	}
	roleListString := fmt.Sprintf("%s", claimMap["authority"])
	roleListString = roleListString[1:len(roleListString)-1]
	result.Roles = strings.Split(roleListString, ",")
	return result,nil
}