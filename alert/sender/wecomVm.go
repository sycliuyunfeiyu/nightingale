package sender

import (
	"github.com/ccfos/nightingale/v6/alert/astats"
	"github.com/ccfos/nightingale/v6/pkg/poster"
	"github.com/toolkits/pkg/logger"
	"html/template"
	"strings"
	"time"

	"github.com/ccfos/nightingale/v6/models"
)

type wecomVmMarkdown struct {
	Content string `json:"content"`
}

type wecomVm struct {
	Msgtype  string          `json:"msgtype"`
	Markdown wecomVmMarkdown `json:"markdown"`
}

type WecomVmSender struct {
	tpl *template.Template
}

func (ws *WecomVmSender) Send(ctx MessageContext) {
	if len(ctx.Users) == 0 || len(ctx.Events) == 0 {
		return
	}
	urls := ws.extract(ctx.Users)
	message := BuildTplMessage(models.WecomVm, ws.tpl, ctx.Events)
	for _, url := range urls {
		body := wecomVm{
			Msgtype: "markdown",
			Markdown: wecomVmMarkdown{
				Content: message,
			},
		}
		wxVmDoSend(url, body, models.WecomVm, ctx.Stats)
	}
}

func (ws *WecomVmSender) extract(users []*models.User) []string {
	urls := make([]string, 0, len(users))
	for _, user := range users {
		if token, has := user.ExtractToken(models.WecomVm); has {
			url := token
			if !strings.HasPrefix(token, "https://") && !strings.HasPrefix(token, "http://") {
				url = "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=" + token
			}
			urls = append(urls, url)
		}
	}
	return urls
}
func wxVmDoSend(url string, body interface{}, channel string, stats *astats.Stats) {
	stats.AlertNotifyTotal.WithLabelValues(channel).Inc()
	//res, code, err := poster.PostJSON(url, time.Second*5, body, 3)

	res, code, err := poster.PostJSONProxy(url, time.Second*5, body, 3)
	if err != nil {
		logger.Errorf("%s_sender: result=fail url=%s code=%d error=%v req:%v response=%s", channel, url, code, err, body, string(res))
		stats.AlertNotifyErrorTotal.WithLabelValues(channel).Inc()
	} else {
		logger.Infof("%s_sender: result=succ url=%s code=%d req:%v response=%s", channel, url, code, body, string(res))
	}
}
