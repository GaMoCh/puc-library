# PUC Library

This app will search for books at [PUC Library](https://portal.pucminas.br/biblioteca) and return its as a PDF file.

## Running

### App

#### Run with Docker image (Recommended)

#### Build

```
docker build -t puc-library .
```

#### Run

```
docker run --rm puc-library > books.pdf
```

#### Run with Go binary (Recommend with chromedp/headless-shell)

```
go run . > books.pdf
```

### Browser with remote debugger

#### Run with [chromedp/headless-shell](https://hub.docker.com/r/chromedp/headless-shell/) Docker image

```
docker run -d -p 9222:9222 --rm --name headless-shell chromedp/headless-shell:98.0.4758.102
```

#### Run with Chrome/Chromium binary

Run Chrome/Chromium binary with flag `--remote-debugging-port=9222`

**Warning**: Can return the error `Printing is not available`

### Available environment variables

- **PS_QUERY**: (Default: "Livros de Teste de Software") - Query that will be submitted at PUC library form.
- **PS_PAGES**: (Default: "5") - Total number of pages that will be searched.
- **PS_TIMEOUT**: (Default: "30") - Seconds Timeout of each step of app execution.
- **PS_REMOTE_TIMEOUT**: (Default: "5") - Seconds Timeout to establish connection with remote browser.
- **PS_HEADLESS**: (Default: "false") - Headless mode when remote browser connection not established.
