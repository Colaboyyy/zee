package gee

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"strconv"
)

var SqlDb *sql.DB
var sqlRes SqlResponse
var sqlStr string

type Cantonese struct {
	Id        int    `json:"id"`
	CharCN    string `json:"charCN"`
	Pronounce string `json:"pronounce"`
}

type SqlResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// DB中间件用
func DbFindProByChar() HandlerFunc {
	return func(ctx *Context) {
		FindPronounceByChar(ctx)
	}
}

func DbInit() {
	// Load接收多个文件名作为参数，如果不传入文件名，默认读取.env文件的内容
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Load .env failed!", err)
	}
	sqlStr = os.Getenv("sqlStr")
	SqlDb, _ = sql.Open("mysql", sqlStr)
}

// FindOneById 根据id查询数据
func FindOneById(c *Context) {
	DbInit()
	id := c.Query("id")
	idInt, _ := strconv.Atoi(id)
	fmt.Println("id:", idInt)
	sqlStr = "select id,pronounce,charCN from cantonese where id=?"
	// 定义接收数据
	var canton Cantonese
	//select上有几个字段，scan将有几个接收
	err := SqlDb.QueryRow(sqlStr, idInt).Scan(&canton.Id, &canton.Pronounce, &canton.CharCN)
	if err != nil {
		fmt.Println("FindOneById failed!", err)
		sqlRes.Code = 1
		sqlRes.Message = "FindOneById查询失败"
		sqlRes.Data = "null"
		//c.JSON(http.StatusOK, sqlRes)
		return
	}
	sqlRes.Code = 0
	sqlRes.Message = "FindOneById查询成功"
	sqlRes.Data = canton
	//c.JSON(http.StatusOK, sqlRes)
	err = SqlDb.Close()
	if err != nil {
		fmt.Println("Mysql Close failed!", err)
		return
	}
}

func FindPronounceByChar(c *Context) {
	DbInit()

	charCN := c.Query("charCN")
	queryStr := "select id,pronounce from cantonese where charCN=?"
	// 定义接收数据
	var canton Cantonese
	//select上有几个字段，scan将有几个接收
	err := SqlDb.QueryRow(queryStr, charCN).Scan(&canton.Id, &canton.Pronounce)
	if err != nil {
		fmt.Println("FindPronounceByChar failed!", err)
		sqlRes.Code = 1
		sqlRes.Message = "FindPronounceByChar 查询失败"
		sqlRes.Data = "null"
		c.JSON(http.StatusOK, sqlRes)
		return
	}

	sqlRes.Code = 0
	sqlRes.Message = "FindPronounceByChar 查询成功"
	sqlRes.Data = canton.Pronounce
	c.SqlRes = sqlRes
	c.JSON(http.StatusOK, sqlRes)
	err = SqlDb.Close()
	if err != nil {
		fmt.Println("Mysql Close failed!", err)
		return
	}
}

func FindMulPronounceByChar(c *Context) {
	DbInit()
	charCN := c.Query("charCN")
	queryStr := "select id,charCN,pronounce from cantonese where charCN=?"
	rows, err := SqlDb.Query(queryStr, charCN)
	if err != nil {
		fmt.Println("findMulData failed!", err)
		sqlRes.Code = 1
		sqlRes.Message = "findMulData 查询失败"
		sqlRes.Data = "null"
		c.JSON(http.StatusOK, sqlRes)
		return
	}
	defer rows.Close()
	// 创建一个切片来存储数据
	resultP := make([]Cantonese, 0)
	for rows.Next() {
		// 定义接收数据
		var canton Cantonese
		rows.Scan(&canton.Id, &canton.CharCN, &canton.Pronounce)
		// 追加到切片中
		resultP = append(resultP, canton)
	}
	sqlRes.Code = 0
	sqlRes.Message = "findMulData 查询成功"
	sqlRes.Data = resultP
	c.JSON(http.StatusOK, sqlRes)
	err = SqlDb.Close()
	if err != nil {
		fmt.Println("Mysql Close failed!", err)
		return
	}
}

func fileIsExist(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			log.Print("File not exist, ", err)
			return false
		} else {
			log.Print("Find file error, ", err)
			return false
		}
	} else {
		return true
	}
}

func FindMp3ForPronounce(c *Context) {
	DbInit()

	pronounce := c.Query("pronounce")
	path := "./gee/static/mp3/" + pronounce + ".mp3"

	if fileIsExist(path) {
		sqlRes.Code = 0
		sqlRes.Message = "FindMp3ForChar 查询成功"
		sqlRes.Data = "assets/mp3/" + pronounce + ".mp3"

	} else {
		sqlRes.Code = 1
		sqlRes.Message = "FindMp3ForPronounce 查询失败"
		sqlRes.Data = nil
	}

	c.JSON(http.StatusOK, sqlRes)
}

// 查询多条数据
func FindMulData(c *Context) {
	DbInit()
	queryStr := "select id,charCN,pronounce from cantonese where charCN=?"

	//select上有几个字段，scan将有几个接收
	rows, err := SqlDb.Query(queryStr)
	if err != nil {
		fmt.Println("findMulData failed!", err)
		sqlRes.Code = 1
		sqlRes.Message = "findMulData 查询失败"
		sqlRes.Data = "null"
		c.JSON(http.StatusOK, sqlRes)
		return
	}
	defer rows.Close()
	// 创建一个切片来存储数据
	resultP := make([]Cantonese, 0)
	for rows.Next() {
		// 定义接收数据
		var canton Cantonese
		rows.Scan(&canton.Id, &canton.CharCN, &canton.Pronounce)
		// 追加到切片中
		resultP = append(resultP, canton)
	}
	sqlRes.Code = 0
	sqlRes.Message = "findMulData 查询成功"
	sqlRes.Data = resultP
	c.JSON(http.StatusOK, sqlRes)
	err = SqlDb.Close()
	if err != nil {
		fmt.Println("Mysql Close failed!", err)
		return
	}
}
