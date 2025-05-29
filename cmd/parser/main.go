package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"log"
	"net/http/cookiejar"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	"google.golang.org/protobuf/proto"
	gmi "mireaattendanceapp/proto/GetAvailableVisitingLogsOfStudent"
	glrsrfsv "mireaattendanceapp/proto/GetLearnRatingScoreReportForStudentInVisitingLog"
)

func callGetAvailableVisitingLogsOfStudent(client *resty.Client) string {
	req := &gmi.GetAvailableVisitingLogsOfStudentRequest{}

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
		log.Fatalf("GetMeInfo request failed: %v", err)
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	UUID := resp.String()[13:49]
	return UUID
}

func Logging(client *resty.Client) {
	client.SetHeader("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 8_4_1; like Mac OS X) AppleWebKit/602.3 (KHTML, like Gecko)  Chrome/49.0.3440.106 Mobile Safari/601.9")
	client.SetRedirectPolicy(resty.FlexibleRedirectPolicy(10))

	jar, _ := cookiejar.New(nil)
	client.SetCookieJar(jar)

	if _, err := client.R().Get("https://attendance-app.mirea.ru/"); err != nil {
		log.Fatal(err)
	}

	//csrf токен
	resp, err := client.R().Get("https://attendance.mirea.ru/api/auth/login?redirectUri=https%3A%2F%2Fattendance-app.mirea.ru&rememberMe=True")

	if err != nil {
		log.Fatal(err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.String()))

	if err != nil {
		log.Fatal(err)
	}

	csrfToken := doc.Find("input[name='csrfmiddlewaretoken']").AttrOr("value", "#")
	nextToken := doc.Find("input[name='next']").AttrOr("value", "#")

	_, err = client.R().
		SetFormData(map[string]string{
			"csrfmiddlewaretoken": csrfToken,
			"login":               "palagnyuk.a.a@edu.mirea.ru",
			"password":            "Aa19102006.",
			"next":                nextToken,
		}).
		SetHeader("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7").
		SetHeader("content-type", "application/x-www-form-urlencoded").
		SetHeader("origin", "https://login.mirea.ru").
		SetHeader("referer", "https://login.mirea.ru/login/?next=/oauth2/v1/authorize/%3Fclient_id%3DRkDSYWk7OPYsJ3KVehRbHRfjxdjIgmiCJ8j8IdvO8%26scope%3Dbasic%26response_type%3Dcode%26redirect_uri%3Dhttps%253A%252F%252Fattendance.mirea.ru%252Fapi%252Fmireaauth%26state%3Doauth_state%253A01970632-7fed-747a-a1c6-7dda25f047f1").
		Post("https://login.mirea.ru/login/")

	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	client := resty.New()

	//Авторизация
	Logging(client)

	//авторизация->gRPC запрос на первый сервис, чтобы получить ID сервис->gRPC запрос, чтобы получить данные RatingScore
	callGetLearnRatingScoreReportForStudentInVisitingLog(client)
}

func callGetLearnRatingScoreReportForStudentInVisitingLog(client *resty.Client) {

	UUID := callGetAvailableVisitingLogsOfStudent(client)
	req := &glrsrfsv.GetScoreAndVisitngRequest{
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

	fmt.Println("Response Body: ", resp)

}
