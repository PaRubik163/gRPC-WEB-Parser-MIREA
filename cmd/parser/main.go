package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net/http/cookiejar"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	"google.golang.org/protobuf/proto"
	respPars "mireaattendanceapp/internal/grpcweb"
	uuid "mireaattendanceapp/proto/GetAvailableVisitingLogsOfStudent"
	respScore "mireaattendanceapp/proto/GetLearnRatingScoreReportForStudentInVisitingLog"
	gmi "mireaattendanceapp/proto/GetMeInfo"
)

func Logging(client *resty.Client, login, password string) error {
	client.SetHeader("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 8_4_1; like Mac OS X) AppleWebKit/602.3 (KHTML, like Gecko)  Chrome/49.0.3440.106 Mobile Safari/601.9")
	client.SetRedirectPolicy(resty.FlexibleRedirectPolicy(10))

	jar, _ := cookiejar.New(nil)
	client.SetCookieJar(jar)

	if _, err := client.R().Get("https://attendance-app.mirea.ru/"); err != nil {
		return errors.New("Ошибка первого GET запроса")
	}

	//csrf токен
	resp, err := client.R().Get("https://attendance.mirea.ru/api/auth/login?redirectUri=https%3A%2F%2Fattendance-app.mirea.ru&rememberMe=True")

	if err != nil {
		return errors.New("Ошибка второго GET запроса")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.String()))

	if err != nil {
		return errors.New("Ошибка создания файла для парсинга страницы")
	}

	csrfToken := doc.Find("input[name='csrfmiddlewaretoken']").AttrOr("value", "#")
	nextToken := doc.Find("input[name='next']").AttrOr("value", "#")

	_, err = client.R().
		SetFormData(map[string]string{
			"csrfmiddlewaretoken": csrfToken,
			"login":               login,
			"password":            password,
			"next":                nextToken,
		}).
		SetHeader("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7").
		SetHeader("content-type", "application/x-www-form-urlencoded").
		SetHeader("origin", "https://login.mirea.ru").
		SetHeader("referer", "https://login.mirea.ru/login/?next=/oauth2/v1/authorize/%3Fclient_id%3DRkDSYWk7OPYsJ3KVehRbHRfjxdjIgmiCJ8j8IdvO8%26scope%3Dbasic%26response_type%3Dcode%26redirect_uri%3Dhttps%253A%252F%252Fattendance.mirea.ru%252Fapi%252Fmireaauth%26state%3Doauth_state%253A01970632-7fed-747a-a1c6-7dda25f047f1").
		Post("https://login.mirea.ru/login/")

	if err != nil {
		return errors.New("Ошибка POST запроса")
	}

	return nil
}

func callGetMeInfo(client *resty.Client) (string, error) {
	req := &gmi.GetMeInfoRequest{
		Url:     "https://attendance-app.mirea.ru/services",
		Version: 3,
	}

	data, err := proto.Marshal(req)

	if err != nil {
		return "", errors.New("Ошибка кодирования")
	}

	var buf bytes.Buffer
	buf.WriteByte(0x00)
	binary.Write(&buf, binary.BigEndian, uint32(len(data)))
	buf.Write(data)

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())

	resp, err := client.R().
		SetHeader("Content-Type", "application/grpc-web-text").
		SetHeader("X-Grpc-Web", "1").
		SetHeader("Origin", "https://attendance-app.mirea.ru").
		SetHeader("Referer", "https://attendance-app.mirea.ru/").
		SetHeader("User-Agent", "grpc-web-javascript/0.1").
		SetHeader("X-Requested-With", "XMLHttpRequest").
		SetHeader("Accept-Encoding", "gzip, deflate, br, zstd").
		SetHeader("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7").
		SetBody(encoded).
		Post("https://attendance.mirea.ru/rtu_tc.rtu_attend.app.UserService/GetMeInfo")

	if err != nil {
		errors.New("Ошибка gRPC запроса GetMeInfo")
	}

	res := "\n" + resp.String()[50:100]
	return res, nil
}

func callGetAvailableVisitingLogsOfStudent(client *resty.Client) (string, error) {
	req := &uuid.GetAvailableVisitingLogsOfStudentRequest{}

	// сериализация protobuf
	data, err := proto.Marshal(req)
	if err != nil {
		return "", errors.New("Ошибка кодирования")
	}

	// gRPC-web frame: 1 байт флага и 4 байта длины
	var buf bytes.Buffer
	buf.WriteByte(0x00)
	binary.Write(&buf, binary.BigEndian, uint32(len(data)))
	buf.Write(data)

	// grpc-web-text требует base64
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())

	// отправка запроса
	resp, err := client.R().
		SetHeader("Content-Type", "application/grpc-web-text").
		SetHeader("X-Grpc-Web", "1").
		SetHeader("Origin", "https://attendance-app.mirea.ru").
		SetHeader("Referer", "https://attendance-app.mirea.ru/").
		SetHeader("User-Agent", "grpc-web-javascript/0.1").
		SetHeader("X-Requested-With", "XMLHttpRequest").
		SetHeader("Accept-Encoding", "gzip, deflate, br, zstd").
		SetHeader("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7").
		SetBody(encoded).
		Post("https://attendance.mirea.ru/rtu_tc.attendance.api.VisitingLogService/GetAvailableVisitingLogsOfStudent")

	if err != nil {
		return "", errors.New("Ошибка gRPC-WEB запроса для взятия ID")
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	UUID := resp.String()[13:49]
	if len(UUID) < 36 {
		return "", errors.New("ID >36 МБ Ошибки в Логине или Пароле")
	}
	return UUID, nil
}

func main() {
	LogPass := map[string]string{
		"palagnyuk.a.a@edu.mirea.ru":  "Aa19102006.",
		"bakyr.m.y@edu.mirea.ru":      "Mert251326Mert@",
		"andreev.a.r1@edu.mirea.ru":   "123EWQasdD!",
		"gorchakov.a.a@edu.mirea.ru":  "gognop-wyzpUp-1zyqru",
		"prygov.k.d@edu.mirea.ru":     "Kirill_200622",
		"eremushkin.g.r@edu.mirea.ru": "JoJo24578!",
		"kogay.a.s@edu.mirea.ru":      "I61s322d73p84_",
		"pavlov.d.d1@edu.mirea.ru":    "1May20060987__",
		"didenko.d.m@edu.mirea.ru":    "Dari_2208371900!",
		"rokhlin.n.a@edu.mirea.ru":    "Student12!mirea",
	}

	//Авторизация
	for l, p := range LogPass {
		client := resty.New()
		err := Logging(client, l, p)

		if err != nil {
			log.Fatal(err)
			continue
		}

		//авторизация->gRPC запрос на GetMeInfo для получения ФИО студента
		FIO, err := callGetMeInfo(client)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(FIO)

		//авторизация->gRPC запрос на первый сервис, чтобы получить ID сервис->gRPC запрос, чтобы получить данные RatingScore
		studyReport := callGetLearnRatingScoreReportForStudentInVisitingLog(client)

		//Конечно, нужно декодировать в структуру из прото, но я пока не понимаю как
		res := respPars.ParseGrpcResponse(studyReport)

		for _, sub := range res {
			subjectName, _ := sub["name"].(string)
			visitsScore, _ := sub["attendance"].(float64)
			semestrScore, _ := sub["current_control"].(float64)
			fmt.Println(subjectName, "--------", visitsScore+semestrScore)
		}
	}

}

func callGetLearnRatingScoreReportForStudentInVisitingLog(client *resty.Client) []byte {

	UUID, err := callGetAvailableVisitingLogsOfStudent(client)

	if err != nil {
		log.Fatal(err)
	}

	req := &respScore.GetScoreAndVisitngRequest{
		Id: UUID,
	}

	// сериализация protobuf
	data, err := proto.Marshal(req)
	if err != nil {
		log.Fatalf("marshal GetMeInfoRequest: %v", err)
	}

	// gRPC-web frame: 1 байт флага и 4 байта длины
	var buf bytes.Buffer
	buf.WriteByte(0x00)
	binary.Write(&buf, binary.BigEndian, uint32(len(data)))
	buf.Write(data)

	// grpc-web-text требует base64
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())

	resp, err := client.R().
		SetHeader("Content-Type", "application/grpc-web-text").
		SetHeader("X-Grpc-Web", "1").
		SetHeader("Origin", "https://attendance-app.mirea.ru").
		SetHeader("Referer", "https://attendance-app.mirea.ru/").
		SetBody(encoded).
		Post("https://attendance.mirea.ru/rtu_tc.attendance.api.LearnRatingScoreService/GetLearnRatingScoreReportForStudentInVisitingLog")

	if err != nil {
		log.Fatal(err)
	}

	return resp.Body()
}
