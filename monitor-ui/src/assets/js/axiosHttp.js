import axios from 'axios'
import {baseURL_config} from './baseURL'
import { getToken } from '@/assets/js/cookies.ts'
export default function ajax (options) {
  const ajaxObj = {
    method: options.method,
    baseURL: baseURL_config,
    url: options.url,
    timeout: 30000,
    params: options.params,
    // params: options.params || '',
    headers: {
      'Content-type': 'application/json;charset=UTF-8',
      'X-Auth-Token': getToken() || null,
      'Authorization': 'Bearer eyJhbGciOiJIUzUxMiJ9.eyJzdWIiOiJhZG1pbiIsImlhdCI6MTY3NTQwOTI4MywidHlwZSI6ImFjY2Vzc1Rva2VuIiwiY2xpZW50VHlwZSI6IlVTRVIiLCJleHAiOjE2NzU0MTA0ODMsImF1dGhvcml0eSI6IltTVVBFUl9BRE1JTixJTVBMRU1FTlRBVElPTl9XT1JLRkxPV19FWEVDVVRJT04sSU1QTEVNRU5UQVRJT05fQVJUSUZBQ1RfTUFOQUdFTUVOVCxNT05JVE9SX01BSU5fREFTSEJPQVJELE1PTklUT1JfTUVUUklDX0NPTkZJRyxNT05JVE9SX0FMQVJNX0NPTkZJRyxNT05JVE9SX0FMQVJNX01BTkFHRU1FTlQsQ09MTEFCT1JBVElPTl9QTFVHSU5fTUFOQUdFTUVOVCxDT0xMQUJPUkFUSU9OX1dPUktGTE9XX09SQ0hFU1RSQVRJT04sQURNSU5fU1lTVEVNX1BBUkFNUyxBRE1JTl9SRVNPVVJDRVNfTUFOQUdFTUVOVCxBRE1JTl9VU0VSX1JPTEVfTUFOQUdFTUVOVCxBRE1JTl9DTURCX01PREVMX01BTkFHRU1FTlQsQ01EQl9BRE1JTl9CQVNFX0RBVEFfTUFOQUdFTUVOVCxBRE1JTl9RVUVSWV9MT0csTUVOVV9BRE1JTl9QRVJNSVNTSU9OX01BTkFHRU1FTlQsTUVOVV9JRENfUkVTT1VSQ0VfUExBTk5JTkcsTUVOVV9DTURCX0FETUlOX0JBU0VfREFUQV9NQU5BR0VNRU5ULE1FTlVfREVTSUdOSU5HX0NJX0lOVEVHUkFURURfUVVFUllfRVhFQ1VUSU9OLE1FTlVfQVBQTElDQVRJT05fREVQTE9ZTUVOVF9ERVNJR04sTUVOVV9ERVNJR05JTkdfQ0lfREFUQV9NQU5BR0VNRU5ULE1FTlVfREVTSUdOSU5HX0NJX0lOVEVHUkFURURfUVVFUllfTUFOQUdFTUVOVCxNRU5VX0lEQ19QTEFOTklOR19ERVNJR04sTUVOVV9BUFBMSUNBVElPTl9BUkNISVRFQ1RVUkVfUVVFUlksTUVOVV9BUFBMSUNBVElPTl9BUkNISVRFQ1RVUkVfREVTSUdOLE1FTlVfREVTSUdOSU5HX0NJX0RBVEFfRU5RVUlSWSxDQVBBQ0lUWV9NT0RFTCxDQVBBQ0lUWV9GT1JFQ0FTVCxBRE1JTl9JVFNfREFOR0VST1VTX0NPTkZJRyxUUkVFVkVOVF9DT05GSUcsVFJFRVZFTlRfTElTVCxNRU5VX0FETUlOX1FVRVJZX0xPRyxkYXRhX3F1ZXJ5X2NpLGRhdGFfcXVlcnlfcmVwb3J0LGRhdGFfcXVlcnlfdmlldyxkYXRhX21nbXRfY2ksZGF0YV9tZ210X3ZpZXcsbW9kZWxfY29uZmlndXJhdGlvbixyZXBvcnRfY29uZmlndXJhdGlvbixiYXNla2V5X2NvbmZpZ3VyYXRpb24sc3lzdGVtX29wZXJhdGlvbl9sb2csQURNSU5fVEVSUkFGT1JNX0RFQlVHLEFETUlOX1NZU1RFTV9XT1JLRkxPV19SRVBPUlQsVEFTS19URU1QTEFURV9HUk9VUF9NQU5BR0VNRU5ULFRBU0tfVEVNUExBVEVfTUFOQUdFTUVOVCxUQVNLX1JFUVVFU1RfTUFOQUdFTUVOVCxUQVNLX1RBU0tfTUFOQUdFTUVOVCxKT0JTX1NFUlZJQ0VfQ0FUQUxPR19NQU5BR0VNRU5ULE1PTklUT1JfQ1VTVE9NX0RBU0hCT0FSRCxNRU5VX0FETUlOX0NNREJfTU9ERUxfTUFOQUdFTUVOVCxKT0JTX1RBU0tfTUFOQUdFTUVOVCxJTVBMRU1FTlRBVElPTl9CQVRDSF9FWEVDVVRJT04sQURNSU5fVEVSUkFGT1JNX0NPTkZJRyxJTVBMRU1FTlRBVElPTl9URVJNSU5BTCxBRE1JTl9DRVJUSUZJQ0FUSU9OLHN0YXRlX21hY2hpbmVfY29uZmlndXJhdGlvbixjaV90ZW1wbGF0ZV9jb25maWd1cmF0aW9uLGdyYXBoX2NvbmZpZyxBRE1JTl9URVJNSU5BTF9BU1NFVCxBRE1JTl9URVJNSU5BTF9BVURJVCxBRE1JTl9URVJNSU5BTF9DT05GSUcsVFJFRVZFTlRfU0VRX0NPTkZJRyxUUkVFVkVOVF9MT0dfUVVFUlksVFJFRVZFTlRfTU9OSVRPUl9RVUVSWSxBUFBfREVWLEpPQlNfVEVNUExBVEVfR1JPVVBfTUFOQUdFTUVOVCxKT0JTX1RFTVBMQVRFX01BTkFHRU1FTlQsSk9CU19SRVFVRVNUX01BTkFHRU1FTlQsQ01EQl9BRE1JTixQUkRfT1BTXSJ9.Dtw_2TH9hIcYgrpdc9LPLOE3I8EQJQBTOGoH8-bAQ3-uOYrlkRNPiLlZaMfXsST-DHbWE_9hx59AYUcd-Ol-Bw'
    },
    // data: JSON.stringify(options.data || '')
    data: JSON.stringify(options.data)
  }
  // 导出请求时增加响应类型
  if (options.url.endsWith('/export')) {
    ajaxObj.responseType = 'blob'
  }
  return window.request ? window.request(ajaxObj) : axios(ajaxObj)
}
