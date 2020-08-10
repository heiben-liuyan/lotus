package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/specs-actors/actors/crypto"
	"github.com/urfave/cli/v2"
)

var contents = ""
var sectorSize = ""

func GetBetweenStr(str, start, end string) string {
	n := strings.Index(str, start)
	if n == -1 {
		n = 0
	}
	n = n + len(start)
	str = string([]byte(str)[n:])
	m := strings.Index(str, end)
	if m == -1 {
		m = len(str)
	}
	str = string([]byte(str)[:m])
	return str
}

func createFile(filepath string) string {
	filepath = filepath + "/create" + fmt.Sprintf("%d", time.Now().Unix()) + ".txt"
	file, err := os.Create(filepath)
	defer file.Close()
	if err != nil {
		println(err)
		return ""
	}
	return filepath
}

func createMiner(uri string, wallet string) error {
	/*wallet = strings.Replace(wallet, " ", "", -1)
	wallet = strings.Replace(wallet, "\r", "", -1)
	wallet = strings.Replace(wallet, "\n", "", -1)*/
	wallet = strings.Replace(wallet, "\r", "", -1)
	/*println("wallet-len",len(wallet))
	println("wallet:",wallet)
	println("创建矿工：",uri+"/mkminer?sectorSize=1024&address="+wallet[0 : 86])*/
	resp1, err1 := http.Get(uri + "/mkminer?sectorSize=" + sectorSize + "&address=" + wallet)

	if err1 != nil {
		println("err1:", err1)
		return nil
	}
	defer resp1.Body.Close()

	u, _ := url.ParseQuery(resp1.Request.URL.RawQuery)
	f := u.Get("f")
	http.Get(uri + "/msgwait?cid=" + f)
	//println("创建矿工消息：",uri+"/msgwaitaddr?cid="+f)

	resp2, err2 := http.Get(uri + "/msgwaitaddr?cid=" + f)
	if err2 != nil {
		println("err2:", err2)
		return nil
	}
	defer resp2.Body.Close()

	body, _ := ioutil.ReadAll(resp2.Body)
	//println(string(body))
	if string(body) == "cid too short" {
		println("钱包", wallet, "创建矿工失败")
		return nil
	}
	bodyStr := GetBetweenStr(string(body), `{"addr":"`, `"}`)
	content := "lotus-storage-miner init --actor=" + bodyStr + " --owner=" + wallet + "\n"

	contents = contents + content
	println("lotus-storage-miner init --owner=" + content)
	return nil
}

var hlmFaucetCmd = &cli.Command{
	Name:  "hlm-faucet",
	Usage: "Manage faucet",
	Subcommands: []*cli.Command{
		hlmMinerCmd,
	},
}

var hlmMinerCmd = &cli.Command{
	Name:  "create-miner",
	Usage: "create new miner",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "uri",
			Usage: "fountain server uri",
			Value: "http://120.77.213.165:7776",
		},
		&cli.IntFlag{
			Name:  "num",
			Usage: "Number of miner which want to created.",
			Value: 1,
		},
		&cli.StringFlag{
			Name:  "import-file",
			Usage: "import wallet address file.",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "export-file",
			Usage: "create miner file.",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "sector-size",
			Usage: "create sector size.",
			Value: "1073741824",
		},
	},
	Action: func(cctx *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		uri := cctx.String("uri")
		num := cctx.Int("num")
		importfile := cctx.String("import-file")
		exportfile := cctx.String("export-file")
		sectorSize = cctx.String("sector-size")
		println("uri:", uri)
		path := createFile(exportfile)
		if len(importfile) != 0 {
			f, err := ioutil.ReadFile(importfile)
			if err != nil {
				println("read fail", err)
			}
			wallet := strings.Split(string(f), "\n")
			num = len(wallet)
		}
		println("num:", num)
		result := make(chan error, num)
		if len(importfile) == 0 {
			api, closer, err := lcli.GetFullNodeAPI(cctx) // TODO: consider storing full node address in config
			if err != nil {
				return err
			}
			defer closer()
			ctx := lcli.ReqContext(cctx)

			v, err := api.Version(ctx)
			if err != nil {
				return err
			}
			fmt.Println("api version:", v)
			// doing create
			for i := 0; i < num; i++ {
				go func() {
					nk, _ := api.WalletNew(ctx, crypto.SigTypeBLS)
					result <- createMiner(uri, nk.String())
				}()
			}
		} else {
			f, err := ioutil.ReadFile(importfile)
			if err != nil {
				println("read fail", err)
			}
			//println("wallets:",string(f))
			wallet := strings.Split(string(f), "\n")
			//println(len(wallet))
			for i := 0; i < len(wallet); i++ {
				go func(n int) {
					if len(wallet[n]) > 0 {
						println("wallet", n, wallet[n])
						result <- createMiner(uri, wallet[n])
					} else {
						result <- nil
					}
				}(i)
			}
		}

		// waiting result
		for i := 0; i < num; i++ {
			err := <-result
			if err != nil {
				println(err.Error())
			}
		}

		data := []byte(contents)
		if ioutil.WriteFile(path, data, 0644) == nil {
			println("写入文件成功:\n", contents)
		}
		println("文件地址：", path)
		runtime.Gosched()
		return nil
	},
}
