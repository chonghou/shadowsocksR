package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"strings"

	"github.com/nadoo/glider/proxy"
	"github.com/v2rayA/shadowsocksR/client"
)

type Params struct {
	Method, Passwd, Address, Port, Obfs, ObfsParam, Protocol, ProtocolParam string
}

func convertDialerURL(params Params) (s string, err error) {
	u, err := url.Parse(fmt.Sprintf(
		"ssr://%v:%v@%v:%v",
		params.Method,
		params.Passwd,
		params.Address,
		params.Port,
	))
	if err != nil {
		return
	}
	q := u.Query()
	if len(strings.TrimSpace(params.Obfs)) <= 0 {
		params.Obfs = "plain"
	}
	if len(strings.TrimSpace(params.Protocol)) <= 0 {
		params.Protocol = "origin"
	}
	q.Set("obfs", params.Obfs)
	q.Set("obfs_param", params.ObfsParam)
	q.Set("protocol", params.Protocol)
	q.Set("protocol_param", params.ProtocolParam)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func main() {
	fmt.Println("Serve on :8118")
	// tcp连接，监听8080端口
	l, err := net.Listen("tcp", ":8118")
	if err != nil {
		log.Panic(err)
	}

	// 死循环，每当遇到连接时，调用handle
	for {
		client, err := l.Accept()
		if err != nil {
			log.Panic(err)
		}
		go handle(client)
	}

}

func handle(clien net.Conn) {
	if clien == nil {
		return
	}
	defer clien.Close()

	log.Printf("remote addr: %v\n", clien.RemoteAddr())

	// 用来存放客户端数据的缓冲区
	var b [1024]byte
	//从客户端获取数据
	n, err := clien.Read(b[:])
	if err != nil {
		log.Println(err)
		return
	}

	var method, URL, address string
	// 从客户端数据读入method，url
	fmt.Sscanf(string(b[:bytes.IndexByte(b[:], '\n')]), "%s%s", &method, &URL)
	hostPortURL, err := url.Parse(URL)
	if err != nil {
		log.Println(err)
		return
	}

	// 如果方法是CONNECT，则为https协议
	if method == "CONNECT" {
		address = hostPortURL.Scheme + ":" + hostPortURL.Opaque
	} else { //否则为http协议
		address = hostPortURL.Host
		// 如果host不带端口，则默认为80
		if strings.Index(hostPortURL.Host, ":") == -1 { //host不带端口， 默认80
			address = hostPortURL.Host + ":80"
		}
	}

	s, err := convertDialerURL(Params{
		Method:        "aes-256-cfb",
		Passwd:        "eIW0Dnk69454e6nSwuspv9DmS201tQ0D",
		Address:       "172.104.161.174",
		Port:          "8099",
		Obfs:          "plain",
		ObfsParam:     "",
		Protocol:      "origin",
		ProtocolParam: "",
	})
	if err != nil {
		log.Fatal(err)
	}
	dia, err := client.NewSSRDialer(s, proxy.Default)
	if err != nil {
		log.Fatal(err)
	}

	//获得了请求的host和port，向服务端发起tcp连接
	server, err := dia.Dial("tcp", address)
	if err != nil {
		log.Println(err)
		return
	}
	//如果使用https协议，需先向客户端表示连接建立完毕
	if method == "CONNECT" {
		fmt.Fprint(clien, "HTTP/1.1 200 Connection established\r\n\r\n")
	} else { //如果使用http协议，需将从客户端得到的http请求转发给服务端
		server.Write(b[:n])
	}

	//将客户端的请求转发至服务端，将服务端的响应转发给客户端。io.Copy为阻塞函数，文件描述符不关闭就不停止
	go io.Copy(server, clien)
	io.Copy(clien, server)
}
