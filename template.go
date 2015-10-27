package tinygo

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
)

// 根据视图路径映射视图模板
var viewsMapper = map[string]*template.Template{}

// compileAllViews 根据tinyConfig.CompilePages设置编译全部视图
func compileAllViews() {
	if tinyConfig.precompile {
		filepath.Walk(tinyConfig.view, func(filePath string, fileInfo os.FileInfo, err error) error {
			if fileInfo != nil && !fileInfo.IsDir() && path.Ext(fileInfo.Name()) == DefaultTemplateExt {
				filePath = generateViewFilePath(filePath)
				if !isLayoutFile(filePath) {
					var tmpl, err = compileView(filePath)
					if err == nil {
						viewsMapper[filePath] = tmpl
					} else {
						fmt.Println(err)
					}
				}
			}
			return nil
		})
	}
}

// compileView 编译单个视图
//  filePath: 相对于tinyConfig.ViewPath的文件路径,分隔符必须为/
func compileView(filePath string) (*template.Template, error) {
	var pathSlice = make([]string, 0, 2)
	var lastFile = filePath
	for lastFile != "" {
		pathSlice = append(pathSlice, getViewFilePath(lastFile))
		lastFile, _ = getLayoutFile(lastFile)
	}
	var tmpl, err = template.ParseFiles(pathSlice...)
	if err == nil {
		var name = filepath.Base(pathSlice[len(pathSlice)-1])
		tmpl = tmpl.Lookup(name)
	}
	return tmpl, err
}

// viewTemplate 返回指定视图的模板
//  filePath:相对于tinyConfig.ViewPath的文件路径,分隔符必须为/
func viewTemplate(filePath string) *template.Template {
	var tmpl, ok = viewsMapper[filePath]
	if !ok {
		tmpl, err := compileView(filePath)
		if err != nil {
			fmt.Println(err)
			return nil
		}
		return tmpl
	}
	return tmpl
}

// partailViewTemplate 返回指定部分视图的模板
//  filePath:相对于tinyConfig.ViewPath的文件路径,分隔符必须为/
func partialViewTemplate(filePath string) *template.Template {
	var tmpl, ok = viewsMapper[filePath]
	if !ok {
		tmpl, err := compileView(filePath)
		if err != nil {
			fmt.Println(err)
			return nil
		}
		return tmpl.Lookup(path.Base(filePath))
	}
	return tmpl.Lookup(path.Base(filePath))
}

// ParseTemplate 分析指定模板,如果模板不存在或者出错,则会返回HttpNotFound
//  w:http响应写入器
//  r:http请求
//  path:相对于tinyConfig.ViewPath的文件路径,分隔符必须为/
//  data:要解析到模板中的数据
func ParseTemplate(w http.ResponseWriter, r *http.Request, path string, data interface{}) {
	var tmpl = viewTemplate(path)
	if tmpl != nil {
		err := tmpl.Execute(w, data)
		if err != nil {
			fmt.Println(err)
			HttpNotFound(w, r)
		}
	}
}

// ParsePartialTemplate 分析指定部分模板,如果模板不存在或者出错,则会返回HttpNotFound
//
// 默认情况下,会首先寻找名为"Content"的模板并执行,如果"Content"模板不存在,则直接执行文件模板
//  w:http响应写入器
//  r:http请求
//  path:相对于tinyConfig.ViewPath的文件路径,分隔符必须为/
//  data:要解析到模板中的数据
func ParsePartialTemplate(w http.ResponseWriter, r *http.Request, path string, data interface{}) {
	var tmpl = partialViewTemplate(path)
	if tmpl != nil {
		content := tmpl.Lookup("Content")
		if content != nil {
			tmpl = content
		}
		err := tmpl.Execute(w, data)
		if err != nil {
			fmt.Println(err)
			HttpNotFound(w, r)
		}
	}
}

// mapStructToMap 将一个结构体所有字段(包括通过组合得来的字段)到一个map中
//  value:结构体的反射值
//  data:存储字段数据的map
func mapStructToMap(value reflect.Value, data map[interface{}]interface{}) {
	if value.Kind() == reflect.Struct {
		for i := 0; i < value.NumField(); i++ {
			var fieldValue = value.Field(i)
			if fieldValue.CanInterface() {
				var fieldType = value.Type().Field(i)
				if fieldType.Anonymous {
					//匿名组合字段,进行递归解析
					mapStructToMap(fieldValue, data)
				} else {
					//非匿名字段
					var fieldName = fieldType.Tag.Get("to")
					if fieldName == "" {
						fieldName = fieldType.Name
					}
					data[fieldName] = fieldValue.Interface()
				}
			}
		}
	}
}
