# GoMatrix
这是一个更加极简版本的go框架，由基础的`net/http`封装而成，大部分参考了gin。

## 初始化

相比于`New`创建新的引擎，更推荐使用`Default`来创建，如果您有更好的日志或者错误恢复方案，也可忽略。

```go
r := GoMatrix.New()
// or
r := GoMatrix.Default()
```

同时，框架支持SSL证书，需要使用`SslNew`来进行，只需将证书位置写在其中即可：

```go
r := GoMatrix.SslNew("./gomatrix.crt","./gomatrix.key")
```

启动，只需要调用Run方法即可，您需要传入您的`ip`、端口号以及期待的最大连接数量，此数值没有默认，可根据情况动态变化。

```go
err := r.Run("localhost","8080",10000)
if err != nil {
    return
}
```

如此，便开启了一个http服务

## Request

> `GoMatrix`提供了`GET`、`POST`、`PUT`、`DELETE`、`PATCH`、`CONNECT`、`OPTIONS`、`TRACE`、`HEAD`等HTTP Request

例：

```go
r := GoMatrix.Default()
r.GET("/", func(c *GoMatrix.Context) {
    c.JSON(http.StatusOK, "hello world")
})
err := r.Run("localhost","8080",10000)
if err != nil {
    return
}
```

## Request参数

#### 路径参数

在路由定义时，定义路径参数需要以`:`或者`*`开头，请注意，以`*`开头，则认为其后的全都是参数变量，获取其参数使用`Param`方法，例如：

```go
r.GET("/:param", func(c *GoMatrix.Context) {
    s := c.Param("param")
    c.String(http.StatusOK,s)
})
```

#### Query参数

使用`Query`方法获取

```go
r.GET("/", func(c *GoMatrix.Context) {
    s := c.Query("param")
    c.String(http.StatusOK,s)
})
```

#### Form参数

使用`PostForm`方法获取

```go
r.POST("/", func(c *GoMatrix.Context) {
    s := c.PostForm("param")
    c.String(http.StatusOK,s)
})
```



## Response

> `GoMatrix`提供了诸多Response达成需要的响应，例如`application/json`

```go
r.GET("/", func(c *GoMatrix.Context) {
    c.JSON(http.StatusOK, "hello world")
})
```

亦或`text/plain`

```go
r.GET("/", func(c *GoMatrix.Context) {
    c.String(http.StatusOK, "hello world")
})
```

亦或者文件

```go
r.GET("/", func(c *GoMatrix.Context) {
    c.Data(http.StatusOK,[]byte("hello world"))
})
```

当然，文件亦可下载

```go
r.GET("/", func(c *GoMatrix.Context) {
	c.Download(Path, fileName)
})
```

## 路由分组

使用方法：

```go
r := GoMatrix.Default()

r.GET("/", func(c *GoMatrix.Context) {
    s := c.Query("param")
    c.String(http.StatusOK,s)
})

api := r.Group("/api")
api.GET("/", func(c *GoMatrix.Context) {
    s := c.Query("param")
    c.String(http.StatusOK,s)
})
```

引擎和分组都可用于创建API

