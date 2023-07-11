package sender

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ccfos/nightingale/v6/models"
	"github.com/ccfos/nightingale/v6/pkg/poster"
	"github.com/ccfos/nightingale/v6/tools"
	"github.com/redis/go-redis/v9"
	"github.com/toolkits/pkg/logger"
	"html/template"
	"io/ioutil"
	"net/http"
	"time"
)

type wecomAppUser struct {
	User     string `json:"user"`
	AppToken string `json:"appToken"`
}
type wecomAppMarkdown struct {
	Content string `json:"content"`
}

type wecomApp struct {
	Msgtype  string        `json:"msgtype"`
	Markdown wecomMarkdown `json:"markdown"`
}
type wecomAppToken struct {
	Name string `json:"name"`
	//Token  string `json:"token"`
	Corpid     string `json:"corpid"`
	Corpsecret string `json:"corpsecret"`
}

type WecomAppSender struct {
	tpl *template.Template
}

func (wa *WecomAppSender) Send(ctx MessageContext) {

	if len(ctx.Users) == 0 || ctx.Rule == nil || ctx.Event == nil {
		return
	}

	message := BuildTplMessage(wa.tpl, ctx.Event)
	urlPostList := wa.extract(ctx.Users, message)

	for _, urlPostMap := range urlPostList {

		wa.doSend(urlPostMap["url"], urlPostMap["msg"], urlPostMap["wecomAccessToken"])
	}

}

//https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=ID&corpsecret=SECRET

func (wa *WecomAppSender) extract(users []*models.User, content string) []map[string]string {

	urlPostList := make([]map[string]string, 0, len(users))

	wecomAppPostTemp := `{
				"touser" : "%s",
				"toparty" : "",
				"totag" : "",
				"msgtype" : "markdown",
				"agentid" : 1,
				"text" : {
				"content" : "%s"
			},
				"safe":0,
				"enable_id_trans": 0,
				"enable_duplicate_check": 0,
				"duplicate_check_interval": 1800
			}`
	for _, user := range users {
		var url string

		if userJson, has := user.ExtractToken(models.WecomApp); has {
			var wecomAppT wecomAppToken

			json.Unmarshal([]byte(userJson), &wecomAppT)
			accessToken, err := wa.getAccessToken(&wecomAppT, false)

			if err != nil {
				fmt.Println("获取getAccessToken： " + err.Error())
				logger.Error("获取getAccessToken： " + err.Error())
			}
			logger.Infof("已获取AccessToken" + accessToken)

			wecomAppPostTempStr := fmt.Sprintf(wecomAppPostTemp, wecomAppT.Name, content)

			url = "https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=" + accessToken
			urlPostList = append(urlPostList, map[string]string{"url": url, "msg": wecomAppPostTempStr, "wecomAccessToken": userJson})
		}
	}
	return urlPostList
}

func (was *WecomAppSender) doSend(url string, body string, wecomAccessToken string) {
	type errCodeJosn struct {
		errcode int    `json:"errcode"`
		errmsg  string `json:"errmsg"`
	}
	var errCode errCodeJosn
	res, code, err := poster.PostJSON(url, time.Second*5, body, 3)
	errJson := json.Unmarshal(res, &errCode)
	if errJson != nil || errCode.errcode != 0 {
		var wecomAppT wecomAppToken

		json.Unmarshal([]byte(wecomAccessToken), &wecomAppT)

		accessToken, _ := was.getAccessToken(&wecomAppT, true)
		url = "https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=" + accessToken
	}
	if err != nil {
		logger.Errorf("wecom_sender: result=fail url=%s code=%d error=%v response=%s", url, code, err, string(res))
	} else {
		logger.Infof("wecom_sender: result=succ url=%s code=%d response=%s", url, code, string(res))
	}
}

func (wa *WecomAppSender) getAccessToken(wat *wecomAppToken, retToken bool) (string, error) {
	ctx := context.Background()
	type accessTokenResp struct {
		errcode      int    `json:"errcode"`
		errmsg       string `json:"errmsg"`
		access_token string `json:"access_token"`
		expires_in   int    `json:"expires_in"`
	}
	//{
	//	"errcode": 0,
	//	"errmsg": "ok",
	//	"access_token": "accesstoken000001",
	//	"expires_in": 7200
	//}

	if accessToken, err := tools.RedisClient.Get(ctx, wat.Corpsecret).Result(); err == redis.Nil || retToken {
		logger.Infof("没有找到reids缓存的accessToken，正在重新获取。。。")

		// 重新获取
		//https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=ID&corpsecret=SECRET
		getAccessTokenUrl := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s", wat.Corpid, wat.Corpsecret)
		resp, err := http.Get(getAccessTokenUrl)
		atr := accessTokenResp{}
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		err = json.Unmarshal(body, &atr)
		if err != nil {
			return "", err
		}
		tools.RedisClient.Set(ctx, wat.Corpsecret, atr.access_token, time.Duration(atr.expires_in)*time.Second)
		return atr.access_token, nil

	} else if err != nil {
		logger.Errorf("企业微信应用链接redis失败 " + err.Error())
		fmt.Println("企业微信应用链接redis失败 " + err.Error())
		return "", err
	} else {
		logger.Infof("命中accessToken缓存，正在返回。。。")

		return accessToken, nil
	}

}
