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
	UserNotFound             = 401003
)

const (
	CourseInvalidInput = 402001
)

const (
	QuestionInvalidInput = 403001
	QuestionNotFound     = 403002
	QuestionBizNotFound  = 403003
)

const (
	EvaluationInvalidInput     = 404001
	EvaluationPermissionDenied = 404002
	EvaluationNotFound         = 404003
)

const (
	CommentInvalidInput = 405001
	CommentNotFound     = 405002
)

const (
	SearchInvalidInput = 406001
)

const (
	GradeRepeatSigning = 407001
	GradeNotSigned     = 407002
)

const (
	StaticInvalidInput     = 408001
	StaticPermissionDenied = 408002
)

const (
	AnswerInvalidInput     = 409001
	AnswerPermissionDenied = 409002
	AnswerNotFound         = 409003
)

const (
	PointsNotEnough = 410001
)

const FeedInvalidInput = 411001
