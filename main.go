package main

import (
	"crypto/cipher"
	"crypto/des"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type UserInfo struct {
	Ok       int   `json:"ok"`
	Userinfo Uinfo `json:"userinfo"`
}

type Uinfo struct {
	Userid   int    `json:"userid"`
	Username string `json:"username"`
	T        string `json:"t"`
	Content  string `json:"content"`
}

type PostInfo struct {
	Id int    `json:"id"`
	T  string `json:"t"`
}

type DelResult struct {
	Ok  int    `json:"ok"`
	Msg string `json:"msg"`
}

// 评论也使用这个结构体
type ReplyResult struct {
	M ReplyItem `json:"M"`
}

type ReplyItem struct {
	Ci int    `json:"Ci"`
	C  string `json:"C"`
	S  int    `json:"S"` //点赞数
}

var userHash string
var uid string
var likeCount = 10

func main() {
	var email string
	var password string
	fmt.Println("请输入账号和密码，输入完成后回车确认。")
	fmt.Println("账号: ")
	_, _ = fmt.Scanln(&email)
	fmt.Println("密码: ")
	_, _ = fmt.Scanln(&password)
	fmt.Println("不删除点赞数达到多少的评论(输入一个数字，默认10)")
	_, _ = fmt.Scanln(&likeCount)

	// email = ""
	// password = ""

	mp := md5.Sum([]byte(password))
	fmt.Println("生成userHash...")
	userHash = d(email, hex.EncodeToString(mp[:]))
	fmt.Println(userHash)
	uInfo := getUserInfo(userHash)
	fmt.Println("生成uid...")
	uid = b(strconv.Itoa(uInfo.Userid))
	getUserPost("")
	getUserReply("")
	getMyComment("")
}

// 获取用户信息
func getUserInfo(s string) Uinfo {
	fmt.Println("获取用户信息...")
	url := "https://my.ruanmei.com/api/User/Get?userHash=" + s + "&extra=4%7Cithome_iphone&appver=692"
	//fmt.Println(url)
	rsp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	if rsp.StatusCode == 200 {
		body, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			fmt.Println("http error: " + err.Error())
		}
		u := &UserInfo{}
		_ = json.Unmarshal(body, u)
		return u.Userinfo
	}
	_ = rsp.Body.Close()
	return Uinfo{}
}

var posts []PostInfo

// 获取发帖
func getUserPost(pid string) {
	url := "https://apiquan.ithome.com/api/post/getuserpost?userid=" + uid + "&logid=" + uid + "&userHash=" + userHash + "&isself=1&pid=" + pid
	fmt.Println(url)
	rsp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	if rsp.StatusCode == 200 {
		body, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			fmt.Println("http error: " + err.Error())
		}
		var ps []PostInfo
		err = json.Unmarshal(body, &ps)
		if err != nil {
			fmt.Println(err)
		}

		if len(ps) == 0 {
			_ = rsp.Body.Close()
			fmt.Println("已获取所有发帖，共 " + strconv.Itoa(len(posts)) + " 条数据...")
			fmt.Println("开始删除...")
			delPost()
			return
		}
		fmt.Println("获取 " + strconv.Itoa(len(ps)) + " 条...")
		pid = strconv.Itoa(ps[len(ps)-1].Id)
		for _, p := range ps {
			// 这个if只是为了不删除我发的标题中有 福利 这两个字的帖子
			if !strings.Contains(p.T, "福利") {
				posts = append(posts, p)
			}
		}
	}
	_ = rsp.Body.Close()
	getUserPost(pid)
}

// 删除帖子
func delPost() {
	var id int
	var t string
	if len(posts) > 0 {
		id = posts[0].Id
		t = posts[0].T
	} else {
		fmt.Println("已删除所有帖子...")
		return
	}

	url := "http://apiquan.ithome.com/api/post/userdel?userHash=" + userHash + "&postId=" + strconv.Itoa(id)
	rsp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		fmt.Println(err)
	}
	r := &DelResult{}
	_ = json.Unmarshal(body, r)
	fmt.Println(t + " -- " + r.Msg)
	posts = posts[1:]
	_ = rsp.Body.Close()
	delPost()
}

var replyList []ReplyItem

// 获取回帖
func getUserReply(rid string) {
	url := "https://apiquan.ithome.com/api/reply/getuserreply?userid=" + uid + "&rid=" + rid
	// fmt.Println(url)
	rsp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	defer rsp.Body.Close()
	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		fmt.Println(err)
	}
	var r []ReplyResult
	_ = json.Unmarshal(body, &r)
	if len(r) == 0 {
		fmt.Println("已获取所有回帖，总数 " + strconv.Itoa(len(replyList)))
		fmt.Println("开始删除...")
		delReply()
		return
	}
	fmt.Println("获取 " + strconv.Itoa(len(r)) + " 回帖")
	for _, rp := range r {
		// 不删除点赞数大于等于likeCount(默认10)的评论
		if rp.M.S < likeCount {
			replyList = append(replyList, ReplyItem{rp.M.Ci, rp.M.C, rp.M.S})
			rid = strconv.Itoa(rp.M.Ci)
		}
	}
	_ = rsp.Body.Close()
	getUserReply(rid)
}

// 删除帖子回复
func delReply() {
	var reply ReplyItem
	if len(replyList) > 0 {
		reply = replyList[0]
		fmt.Println(reply)
	} else {
		fmt.Println("已删除所有回帖...")
		return
	}
	url := "http://apiquan.ithome.com/api/reply/userdel?userHash=" + userHash + "&replyId=" + strconv.Itoa(reply.Ci)
	fmt.Println(url)
	rsp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	defer rsp.Body.Close()
	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		fmt.Println(err)
	}
	r := &DelResult{}
	_ = json.Unmarshal(body, r)
	fmt.Println(r.Msg)
	if strings.Contains(r.Msg, "上限") {
		fmt.Println("达到删除上限，请明天继续...")
		return
	}
	fmt.Println(strconv.Itoa(reply.Ci) + " -- " + r.Msg)
	replyList = replyList[1:]
	delReply()
}

var commentList []ReplyItem
var firstGetComment bool
var erlou bool
var erlouFirst bool

// 获取评论
func getMyComment(cid string) {
	var url string
	if erlou {
		if erlouFirst {
			url = "https://dyn.ithome.com/api/comment/getusercomment?userid=" + uid + "&userHash=" + userHash + "&isself=1&lou=2&lessthanid=" + cid
		} else {
			erlouFirst = true
			url = "https://dyn.ithome.com/api/comment/getusercomment?userid=" + uid + "&userHash=" + userHash + "&isself=1&lou=2"
		}
	} else {
		if firstGetComment {
			url = "https://dyn.ithome.com/api/comment/getusercomment?userid=" + uid + "&userHash=" + userHash + "&isself=1&lessthanid=" + cid
		} else {
			firstGetComment = true
			url = "https://dyn.ithome.com/api/comment/getusercomment?userid=" + uid + "&userHash=" + userHash + "&isself=1"
		}
	}
	fmt.Println(url)
	rsp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	defer rsp.Body.Close()
	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		fmt.Println(err)
	}
	var r []ReplyResult
	_ = json.Unmarshal(body, &r)
	if len(r) == 0 {
		if erlou {
			fmt.Println("已获取所有评论，评论总数: " + strconv.Itoa(len(commentList)))
			fmt.Println("开始删除...")
			delComment()
			return
		} else {
			erlou = true
			getMyComment("")
			return
		}
	}
	for _, rp := range r {
		// 不删除点赞数大于等于likeCount(默认10)的评论
		if rp.M.S < likeCount {
			commentList = append(commentList, ReplyItem{rp.M.Ci, rp.M.C, rp.M.S})
			cid = strconv.Itoa(rp.M.Ci)
		}
	}
	fmt.Println("获取 " + strconv.Itoa(len(r)) + " 条评论")
	getMyComment(cid)
}

// 删除评论
func delComment() {
	var reply ReplyItem
	if len(commentList) > 0 {
		reply = commentList[0]
	} else {
		fmt.Println("已删除所有评论...")
		return
	}
	url := "https://api.ithome.com/api/comment/userdc?userHash=" + userHash + "&cid=" + b(strconv.Itoa(reply.Ci))
	fmt.Println(url)
	rsp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	defer rsp.Body.Close()
	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		fmt.Println(err)
	}
	r := &DelResult{}
	_ = json.Unmarshal(body, r)
	fmt.Println(r.Msg)
	// 好像没有上限限制
	if strings.Contains(r.Msg, "上限") {
		fmt.Println("删除达到上限，请明天继续...")
		return
	}
	commentList = commentList[1:]
	delComment()
}

/*
a b c d 是用于加密账号密码的方法
*/
func a(p []byte) string {
	var result string
	for _, b := range p {
		s := hex.EncodeToString([]byte{b})
		if len(s) == 1 {
			result += "0" + s
		} else {
			result += s
		}
	}
	return result
}

func b(s string) string {
	key := "(#i@x*l%"
	var block cipher.Block
	block, _ = des.NewCipher([]byte(key))

	l := len(s)
	if l < 8 {
		l = 8 - l
	} else {
		l %= 8
		if l != 0 {
			l = 8 - l
		} else {
			l = 0
		}
	}

	for i := 0; i < l; i++ {
		s += "\000"
	}

	ba1 := []byte(s)
	ba2 := ba1
	if len(ba2)%8 != 0 {
		copy(ba1, ba2)
	}

	out := make([]byte, len(ba1))
	dst := out
	bs := block.BlockSize()
	for len(ba1) > 0 {
		block.Encrypt(dst, ba1[:bs])
		ba1 = ba1[bs:]
		dst = dst[bs:]
	}
	return a(out)
}

func c(s string) string {
	key := "(#i@x*l%"
	var block cipher.Block
	block, _ = des.NewCipher([]byte(key))
	l := len(s)
	if l < 8 {
		l = 8 - l
	} else {
		l %= 8
		if l != 0 {
			l = 8 - l
		} else {
			l = 0
		}
	}
	for i := 0; i < l; i++ {
		s += "\000"
	}

	ba1 := []byte(s)
	out := make([]byte, len(ba1))
	dst := out
	bs := block.BlockSize()
	for len(ba1) > 0 {
		block.Encrypt(dst, ba1[:bs])
		ba1 = ba1[bs:]
		dst = dst[bs:]
	}
	return a(out)
}

func d(e string, p string) string {
	s := e + "\f" + p
	return c(s)
}
