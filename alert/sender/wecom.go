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

type wecomMarkdown struct {
	Content string `json:"content"`
}

type wecom struct {
	Msgtype  string        `json:"msgtype"`
	Markdown wecomMarkdown `json:"markdown"`
}

var (
	_ CallBacker = (*WecomSender)(nil)
)

type WecomSender struct {
	tpl *template.Template
}

func (ws *WecomSender) CallBack(ctx CallBackContext) {
	if len(ctx.Events) == 0 || len(ctx.CallBackURL) == 0 {
		return
	}

	message := BuildTplMessage(models.Wecom, ws.tpl, ctx.Events)
	body := wecom{
		Msgtype: "markdown",
		Markdown: wecomMarkdown{
			Content: message,
		},
	}

	doSendAndRecord(ctx.Ctx, ctx.CallBackURL, ctx.CallBackURL, body, "callback", ctx.Stats, ctx.Events)
}

func (ws *WecomSender) Send(ctx MessageContext) {
	if len(ctx.Users) == 0 || len(ctx.Events) == 0 {
		return
	}
	urls, tokens := ws.extract(ctx.Users)
	message := BuildTplMessage(models.Wecom, ws.tpl, ctx.Events)
	for i, url := range urls {
		body := wecom{
			Msgtype: "markdown",
			Markdown: wecomMarkdown{
				Content: message,
			},
		}
		doSendAndRecord(ctx.Ctx, url, tokens[i], body, models.Wecom, ctx.Stats, ctx.Events)
	}
}

func (ws *WecomSender) extract(users []*models.User) ([]string, []string) {
	urls := make([]string, 0, len(users))
	tokens := make([]string, 0, len(users))
	for _, user := range users {
		if token, has := user.ExtractToken(models.Wecom); has {
			url := token
			if !strings.HasPrefix(token, "https://") && !strings.HasPrefix(token, "http://") {
				url = "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=" + token
			}
			urls = append(urls, url)
			tokens = append(tokens, token)
		}
	}
	return urls, tokens
}
func wxDoSend(url string, body interface{}, channel string, stats *astats.Stats) {
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
