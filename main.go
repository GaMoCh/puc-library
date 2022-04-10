package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

type Book struct {
	Index int
	Title string
	Info  string
}

const booksPerPage = 20

var (
	query         = "Livros de Teste de Software"
	remoteBrowser = "localhost:9222"

	timeout = time.Second * 30
	pages   = 5
)

var (
	infoLog  = log.New(os.Stderr, "[ INFO ] ", 0)
	errorLog = log.New(os.Stderr, "[ ERROR ] ", 0)
)

func exit() {
	if r := recover(); r != nil {
		os.Exit(1)
	}
}

func main() {
	defer exit()

	if queryEnv := os.Getenv("PS_QUERY"); queryEnv != "" {
		query = queryEnv
	}
	if remoteBrowserEnv := os.Getenv("PS_REMOTE"); remoteBrowserEnv != "" {
		remoteBrowser = remoteBrowserEnv
	}

	if pagesEnv, _ := strconv.Atoi(os.Getenv("PS_PAGES")); pagesEnv != 0 {
		pages = pagesEnv
	}
	if timeoutEnv, _ := strconv.Atoi(os.Getenv("PS_TIMEOUT")); timeoutEnv != 0 {
		timeout = time.Second * time.Duration(timeoutEnv)
	}

	timer := time.AfterFunc(timeout, func() {
		defer exit()
		errorLog.Panic("Timeout - Exiting")
	})

	headless, _ := strconv.ParseBool(os.Getenv("PS_HEADLESS"))
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
	)

	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	infoLog.Print("Connecting with remote browser")
	if debuggerUrl := getDebugURL(remoteBrowser); debuggerUrl != "" {
		ctx, cancel = chromedp.NewRemoteAllocator(ctx, debuggerUrl)
		defer cancel()
	} else {
		infoLog.Print("Remote connection not established")
		infoLog.Print("Using local browser binary")
	}

	ctx, cancel = chromedp.NewContext(ctx, chromedp.WithLogf(errorLog.Printf))
	defer cancel()

	timer.Reset(timeout)
	infoLog.Print("Opening browser")
	if err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate("http://portal.pucminas.br/biblioteca/index_padrao.php"),
		chromedp.SendKeys("[name=uquery]", query, chromedp.ByQuery),
		chromedp.Click("[type=submit]", chromedp.ByQuery),
	}); err != nil {
		errorLog.Panic(err)
	}
	infoLog.Printf("Searching for \"%s\"", query)

	newTarget := chromedp.WaitNewTarget(ctx, func(info *target.Info) bool {
		return info.URL != ""
	})

	oldCtx := ctx
	ctx, cancel = chromedp.NewContext(ctx, chromedp.WithTargetID(<-newTarget))
	defer cancel()

	if err := page.Close().Do(
		cdp.WithExecutor(oldCtx, chromedp.FromContext(oldCtx).Target),
	); err != nil {
		panic(err)
	}

	books := make([]Book, 0, booksPerPage*pages)
	for pageIndex := 1; pageIndex <= pages; pageIndex++ {
		infoLog.Printf("Extracting books from page %d", pageIndex)
		timer.Reset(timeout)

		var hasNextPage bool
		pageBooks := make([]Book, 0, booksPerPage)
		if err := chromedp.Run(ctx, chromedp.Tasks{
			chromedp.WaitVisible(".result-list"),
			chromedp.Evaluate(`
				document.querySelectorAll('.standard-view-style').forEach(node => node.removeAttribute('class'));
				document.querySelectorAll('.display-info [class]').forEach(node => node.remove());
				[...document.querySelectorAll('.result-list-li:not(.video-panel-li)')].map(node => ({
					index: +node.querySelector('.record-index').textContent.trim().match(/\d+/)[0],
					title: node.querySelector('.title-link').textContent.trim(),
					info: node.querySelector('.display-info').textContent.trim(),
				}));`,
				&pageBooks,
			),
			chromedp.Evaluate("$=document.querySelector('.next'),$?.click(),!!$", &hasNextPage),
		}); err != nil {
			errorLog.Panic(err)
		}

		books = append(books, pageBooks...)
		if !hasNextPage {
			infoLog.Print("No more pages to search")
			break
		}
	}

	html := fmt.Sprintf("<h2 id=\"books-quantity\">Total de livros: %d</h2><hr/>", len(books))
	for _, book := range books {
		html += fmt.Sprintf(
			"<h3>%d. %s</h3><p>%s</p><hr/>",
			book.Index,
			book.Title,
			book.Info,
		)
	}

	script := fmt.Sprintf(
		"document.write('%s');",
		strings.ReplaceAll(html, "'", "\\'"),
	)

	timer.Reset(timeout)
	infoLog.Print("Writing PDF")
	if err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Evaluate(script, nil),
		chromedp.WaitVisible("#books-quantity", chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().Do(ctx)
			if err != nil {
				return err
			}
			if _, err = os.Stdout.Write(buf); err != nil {
				return err
			}

			return nil
		}),
	}); err != nil {
		if e, ok := err.(*cdproto.Error); ok && e.Code == -32000 {
			errorLog.Panic(e.Message + " - Use chromedp/headless-shell")
		} else {
			errorLog.Panic(err)
		}
	}
	infoLog.Print("Done")
}

func getDebugURL(address string) string {
	remoteTimeout := time.Second * 5
	if remoteTimeoutEnv, _ := strconv.Atoi(os.Getenv("PS_REMOTE_TIMEOUT")); remoteTimeoutEnv != 0 {
		remoteTimeout = time.Second * time.Duration(remoteTimeoutEnv)
	}

	client := http.Client{
		Timeout: remoteTimeout,
	}

	response, err := client.Get("http://" + address + "/json/version")
	if err != nil {
		return ""
	}

	var payload map[string]interface{}
	if err = json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return ""
	}

	return payload["webSocketDebuggerUrl"].(string)
}
