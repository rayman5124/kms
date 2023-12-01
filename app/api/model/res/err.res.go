package res

type ErrRes struct {
	Status    int      `json:"status"`
	Timestamp string   `json:"timestamp"`
	Method    string   `json:"method"`
	Path      string   `json:"path"`
	Message   []string `json:"message"`
}

// func ProcessErrRes(errWrap *errutil.ErrWrap, ctx *fiber.Ctx) error {
// 	switch errWrap.Code {
// 	case 400:
// 		return ctx.Status(fiber.StatusBadRequest).JSON(&ClientErrRes{
// 			Status:    fiber.StatusBadRequest,
// 			Timestamp: timeutil.FormatNow(),
// 			Method:    ctx.Method(),
// 			Path:      ctx.Path(),
// 			Message:   strings.Split(errWrap.Message, "\n"),
// 		})
// 	case 404:
// 		return ctx.Status(fiber.StatusNotFound).JSON(&ClientErrRes{
// 			Status:    fiber.StatusNotFound,
// 			Timestamp: timeutil.FormatNow(),
// 			Method:    ctx.Method(),
// 			Path:      ctx.Path(),
// 			Message:   []string{"page not found"},
// 		})
// 	case 500:
// 		fmt.Printf("%v\n", errWrap.Message)
// 		return fiber.ErrInternalServerError
// 	default:
// 		fmt.Printf("%v\n", errWrap.Message)
// 		return fiber.ErrInternalServerError
// 	}
// }
