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

type wecomMidMarkdown struct {
	Content string `json:"content"`
}

type wecomMid struct {
	Msgtype  string           `json:"msgtype"`
	Markdown wecomMidMarkdown `json:"markdown"`
}

type WecomMidSender struct {
	tpl *template.Template
}

func (ws *WecomMidSender) Send(ctx MessageContext) {
	if len(ctx.Users) == 0 || len(ctx.Events) == 0 {
		return
	}
	urls := ws.extract(ctx.Users)
	message := BuildTplMessage(models.WecomMid, ws.tpl, ctx.Events)
	for _, url := range urls {
		body := wecomMid{
			Msgtype: "markdown",
			Markdown: wecomMidMarkdown{
				Content: message,
			},
		}
		wxMidDoSend(url, body, models.WecomMid, ctx.Stats)
	}
}

func (ws *WecomMidSender) extract(users []*models.User) []string {
	urls := make([]string, 0, len(users))
	for _, user := range users {
		if token, has := user.ExtractToken(models.WecomMid); has {
			url := token
			if !strings.HasPrefix(token, "https://") && !strings.HasPrefix(token, "http://") {
				url = "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=" + token
			}
			urls = append(urls, url)
		}
	}
	return urls
}
func wxMidDoSend(url string, body interface{}, channel string, stats *astats.Stats) {
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
