package errs

const (
	// InternalServerError 一个非常含糊的错误码。代表系统内部错误
	InternalServerError = 500001
)

// User 部分，模块代码使用 01
const (
	// UserInvalidInput 一个非常含糊的错误码，代表用户相关的API参数不对
	UserInvalidInput = 401001

	// UserInvalidSidOrPassword 用户输入的学号或者密码不对
	UserInvalidSidOrPassword = 401002
)

const (
	CourseInvalidInput = 402001
)

const (
	QuestionBizNotFound = 403001
)

const (
	EvaluationInvalidInput     = 404001
	EvaluationPermissionDenied = 404002
)

const (
	CommentInvalidInput = 405001
)
