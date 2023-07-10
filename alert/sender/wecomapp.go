package sender

import (
	"encoding/json"
	"fmt"
	"github.com/ccfos/nightingale/v6/models"
	"github.com/ccfos/nightingale/v6/pkg/poster"
	"github.com/toolkits/pkg/logger"
	"html/template"
	"strings"
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

		wa.doSend(urlPostMap["url"], urlPostMap["msg"])
	}

}

func (wa *WecomAppSender) extract(users []*models.User, content string) []map[string]string {

	urlPostList := make([]map[string]string, 0, len(users))
	type wecomAppToken struct {
		Name  string `json:"name"`
		Token string `json:"token"`
	}

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
		var wecomAppT wecomAppToken
		var url string

		if userJson, has := user.ExtractToken(models.WecomApp); has {
			json.Unmarshal([]byte(userJson), &wecomAppT)
			wecomAppPostTempStr := fmt.Sprintf(wecomAppPostTemp, wecomAppT.Name, content)

			if !strings.HasPrefix(wecomAppT.Token, "https://") {
				url = "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=" + wecomAppT.Token
			}

			urlPostList = append(urlPostList, map[string]string{"url": url, "msg": wecomAppPostTempStr})
		}
	}
	return urlPostList
}

func (was *WecomAppSender) doSend(url string, body string) {
	res, code, err := poster.PostJSON(url, time.Second*5, body, 3)
	if err != nil {
		logger.Errorf("wecom_sender: result=fail url=%s code=%d error=%v response=%s", url, code, err, string(res))
	} else {
		logger.Infof("wecom_sender: result=succ url=%s code=%d response=%s", url, code, string(res))
	}
}
