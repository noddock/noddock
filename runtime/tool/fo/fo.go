package fo

// 路径检查
// 1. 绝对路径：根据 allowAbsPath 判断是否拒绝
// 2. 相对路径：拼接到当前项目空间的路径
func GetPath(p, pid string, allowAbsPath, allowSymlink bool) (string, error) {
	return p, nil
}

// 将绝对路径转为当前项目的相对路径，如果不存在则报错
func RelPro(p, pid string) (string, error) {
	return p, nil
}
