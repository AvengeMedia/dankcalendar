package ipc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/AvengeMedia/dankcalendar/core/internal/accounts"
	"github.com/AvengeMedia/dankcalendar/core/internal/oauth"
	"github.com/AvengeMedia/dankgo/log"
)

func HandleAccounts(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	switch req.Method {
	case "accounts.list":
		handleAccountsList(ctx, w, req, deps)
	case "accounts.providers":
		Respond(w, req.ID, accounts.AvailableProviders())
	case "accounts.google.setupGuide":
		Respond(w, req.ID, accounts.GoogleSetupSteps())
	case "accounts.google.start":
		handleGoogleStart(ctx, w, req, deps)
	case "accounts.google.complete":
		handleGoogleComplete(ctx, w, req, deps)
	case "accounts.google.reauth":
		handleGoogleReauth(ctx, w, req, deps)
	case "accounts.google.cancel", "accounts.microsoft.cancel":
		handleFlowCancel(w, req, deps)
	case "accounts.microsoft.setupGuide":
		Respond(w, req.ID, accounts.MicrosoftSetupSteps())
	case "accounts.microsoft.start":
		handleMicrosoftStart(ctx, w, req, deps)
	case "accounts.microsoft.complete":
		handleMicrosoftComplete(ctx, w, req, deps)
	case "accounts.microsoft.reauth":
		handleMicrosoftReauth(ctx, w, req, deps)
	case "accounts.caldav.add":
		handleCalDAVAdd(ctx, w, req, deps)
	case "accounts.ical.add":
		handleICalAdd(ctx, w, req, deps)
	case "accounts.local.add":
		handleLocalAdd(ctx, w, req, deps)
	case "accounts.evolution.add":
		handleEvolutionAdd(ctx, w, req, deps)
	case "accounts.delete":
		handleAccountsDelete(ctx, w, req, deps)
	case "accounts.refresh":
		handleAccountsRefresh(ctx, w, req, deps)
	case "accounts.changed":
		publishAccountsChanged(deps, ParamString(req.Params, "accountId"))
		Respond(w, req.ID, map[string]any{"published": true})
	default:
		RespondError(w, req.ID, "unknown accounts method: "+req.Method)
	}
}

func handleAccountsList(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	items, err := deps.Repo.ListAccounts(ctx)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	out := mapAccounts(items)
	for i, acc := range items {
		out[i]["authorized"] = accounts.Authorized(ctx, deps.Secrets, acc)
	}
	Respond(w, req.ID, out)
}

func handleGoogleReauth(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	accountID := ParamString(req.Params, "accountId")
	if accountID == "" {
		RespondError(w, req.ID, "accountId is required")
		return
	}

	creds, err := accounts.GoogleAppCreds(ctx, deps.Secrets, accountID)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	redirect, err := redirectURL(deps.HTTPAddr)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	brokerFlow, err := oauth.StartBrokerFlow(deps.Broker, redirect)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	flow, err := oauth.NewGoogleFlow(creds, brokerFlow)
	if err != nil {
		brokerFlow.Close()
		RespondError(w, req.ID, err.Error())
		return
	}

	deps.Flows.Register(flow)
	Respond(w, req.ID, map[string]any{
		"state":   flow.State(),
		"authUrl": flow.AuthURL(),
	})
}

type googleStartParams struct {
	DisplayName  string `json:"displayName"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

func parseGoogleStartParams(p map[string]any) (googleStartParams, error) {
	out := googleStartParams{
		DisplayName:  strings.TrimSpace(ParamString(p, "displayName")),
		ClientID:     strings.TrimSpace(ParamString(p, "clientId")),
		ClientSecret: strings.TrimSpace(ParamString(p, "clientSecret")),
	}
	creds, err := accounts.ResolveGoogleClient(oauth.GoogleAppCredentials{
		ClientID:     out.ClientID,
		ClientSecret: out.ClientSecret,
	})
	if err != nil {
		return out, err
	}
	out.ClientID = creds.ClientID
	out.ClientSecret = creds.ClientSecret
	return out, nil
}

func handleGoogleStart(_ context.Context, w *ConnWriter, req Request, deps Deps) {
	params, err := parseGoogleStartParams(req.Params)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	redirect, err := redirectURL(deps.HTTPAddr)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	brokerFlow, err := oauth.StartBrokerFlow(deps.Broker, redirect)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	creds := oauth.GoogleAppCredentials{
		ClientID:     params.ClientID,
		ClientSecret: params.ClientSecret,
	}

	flow, err := oauth.NewGoogleFlow(creds, brokerFlow)
	if err != nil {
		brokerFlow.Close()
		RespondError(w, req.ID, err.Error())
		return
	}

	deps.Flows.Register(flow)

	Respond(w, req.ID, map[string]any{
		"state":   flow.State(),
		"authUrl": flow.AuthURL(),
	})
}

func handleFlowCancel(w *ConnWriter, req Request, deps Deps) {
	state := ParamString(req.Params, "state")
	if state == "" {
		RespondError(w, req.ID, "state is required")
		return
	}
	cancelled := deps.Flows.Cancel(state)
	Respond(w, req.ID, map[string]any{"cancelled": cancelled})
}

func handleGoogleComplete(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	state := ParamString(req.Params, "state")
	if state == "" {
		RespondError(w, req.ID, "state is required")
		return
	}

	pending, ok := deps.Flows.Take(state)
	if !ok {
		RespondError(w, req.ID, "no pending google flow for that state")
		return
	}
	flow, ok := pending.(*oauth.GoogleFlow)
	if !ok {
		RespondError(w, req.ID, "pending flow for that state is not a google flow")
		return
	}

	waitCtx, cancel := context.WithTimeout(ctx, 6*time.Minute)
	defer cancel()

	tok, err := flow.Wait(waitCtx, 5*time.Minute)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	res, err := accounts.FinishGoogle(ctx, deps.Repo, deps.Secrets, flow, tok)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	publishAccountsChanged(deps, res.AccountID)
	Respond(w, req.ID, map[string]any{
		"accountId":   res.AccountID,
		"email":       res.DisplayName,
		"displayName": res.DisplayName,
	})
	kickAccountSync(deps, res.AccountID)
}

func handleAccountsDelete(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	accountID := ParamString(req.Params, "accountId")
	if accountID == "" {
		RespondError(w, req.ID, "accountId is required")
		return
	}

	if err := accounts.Delete(ctx, deps.Repo, deps.Secrets, accountID); err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	publishAccountsChanged(deps, accountID)
	Respond(w, req.ID, map[string]any{"deleted": true})
}

func handleAccountsRefresh(ctx context.Context, w *ConnWriter, req Request, deps Deps) {
	if deps.Sync == nil {
		RespondError(w, req.ID, "sync engine not available")
		return
	}

	accountID := ParamString(req.Params, "accountId")
	if accountID == "" {
		go func() {
			if err := deps.Sync.SyncAll(context.Background()); err != nil {
				log.Warnf("manual sync all: %v", err)
			}
		}()
		Respond(w, req.ID, map[string]any{"started": true, "all": true})
		return
	}

	acc, err := deps.Repo.GetAccount(ctx, accountID)
	if err != nil {
		RespondError(w, req.ID, err.Error())
		return
	}

	go func() {
		if err := deps.Sync.SyncAccount(context.Background(), acc); err != nil {
			log.Warnf("manual sync %s: %v", accountID, err)
		}
	}()

	Respond(w, req.ID, map[string]any{"started": true})
}

func redirectURL(httpAddr string) (string, error) {
	return redirectURLForHost(httpAddr, "", "/oauth/callback")
}

// hostOverride forces the loopback host name; Azure only port-relaxes
// redirect URIs registered as http://localhost, and matches the path
// exactly — Microsoft flows use "/" so users can register the bare
// http://localhost.
func redirectURLForHost(httpAddr, hostOverride, path string) (string, error) {
	if httpAddr == "" {
		return "", errors.New("daemon http server is not listening; cannot complete oauth flow")
	}

	host, port, err := splitHostPort(httpAddr)
	if err != nil {
		return "", err
	}

	switch host {
	case "0.0.0.0", "::", "":
		host = "127.0.0.1"
	}
	if hostOverride != "" {
		host = hostOverride
	}

	u := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   path,
	}
	return u.String(), nil
}

func splitHostPort(addr string) (string, string, error) {
	i := strings.LastIndex(addr, ":")
	if i < 0 {
		return "", "", fmt.Errorf("invalid http addr %q", addr)
	}
	return addr[:i], addr[i+1:], nil
}

func publishAccountsChanged(deps Deps, accountID string) {
	if deps.Bus == nil {
		return
	}
	deps.Bus.Publish("accounts", map[string]any{
		"type":      "changed",
		"accountId": accountID,
	})
}
