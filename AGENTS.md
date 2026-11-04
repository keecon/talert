# talert 에이전트 지침

로그 파일을 필터링해 Slack 웹훅으로 알리는 Go CLI다. 응답과 작업 보고는 한국어로, CLI 도움말은 영어로 작성한다.

프로젝트 지침은 이 파일에서 관리한다. `CLAUDE.md`에는 `@AGENTS.md` 참조만 둔다.

## 구조와 변경 범위

- `cmd/talert/`는 CLI 정의, `internal/parser.go`는 필터링, `internal/watcher.go`는 파일 감시, `internal/notifier.go`는 알림 전송을 담당한다. 웹훅 페이로드는 `internal/slack/`에 있다.
- 기존 함수와 패턴을 재사용하고 요청에 필요한 최소 범위만 수정한다. 구조 정리와 동작 변경은 별도 커밋으로 분리한다.
- 명시적 요청 없이 CLI 옵션 이름·별칭·환경변수·기본값을 변경하지 않는다. CLI 변경 시 관련 도움말도 함께 갱신한다.

## Go와 검증

- `go.mod`의 `go`는 지원 중인 두 안정 계열 중 낮은 계열을 최소 요구 버전으로 삼고, `toolchain`은 최신 안정판으로 갱신한다. 이 값들은 자동 갱신되지 않으므로 버전 변경 시 `go.work`도 함께 맞춘다. CI는 `oldstable`·`stable` 모두 빌드·테스트·린트하고, 릴리스는 `stable`로 빌드한다.
- Mage·린터·tparse는 `go.mod`의 `tool` 선언으로 관리한다. PATH에 별도 설치한 도구보다 `go tool mage`, `go tool golangci-lint`, `go tool tparse`를 사용한다.
- 코드 변경 후 기본 검증은 `go tool mage test build`다. Mage 빌드는 Git 태그 또는 `GITHUB_TAG`에서 버전을 얻는다. 동시성·종료 처리 변경 시 `go test -race ./... -timeout 90s`와 `go vet ./...`도 실행한다. 문서만 바꿨다면 참조와 내용 확인으로 검증한다.
- 하위 Go 검증에는 해당 Go 실행 파일과 `GOTOOLCHAIN=local GOWORK=off`를 사용해 최신 도구로 자동 전환되지 않도록 한다. 린터는 CI처럼 최신 Go로 빌드된 바이너리를 사용한다.
- 새 의존성 추가는 사용자 승인 후 진행한다. 의존성 변경 시 `go mod tidy`를 실행하고 `go.mod`·`go.sum` 및 변경된 워크스페이스 파일을 함께 검토한다.

## 유지해야 할 동작

- 로그 정규식은 최소 두 캡처 그룹을 요구한다. 첫 그룹은 레벨, 두 번째는 메시지다. 레벨은 정확히 일치시키고, 제외 문자열은 로그 줄 전체에서 부분 문자열로 검색한다. 두 비교 모두 대소문자를 구분한다.
- `--level`·`--exclude`·`--webhook-owner`는 반복 옵션과 쉼표 구분을 지원하며 환경변수도 쉼표로 구분한다. `watch`는 파일 끝부터 새 줄을 감시하고, `test`는 파일을 한 번 읽어 일치하는 줄을 표시한다.
- 웹훅 전송 성공 후에만 중복 억제 기록을 남긴다. HTTP 실패 처리, 리다이렉트 차단, 재시도를 포함한 전체 요청 시간 제한을 유지한다. 관련 검증은 `internal/notifier_*_test.go`에 있다.
- 감시 종료 시 대기 알림 배출과 제한 시간 후 취소, 파일 재열기와 오류 전달을 유지한다. 스택 수집의 시간·크기 제한, UTF-8 보존 및 다음 로그 경계를 훼손하지 않는다. 관련 검증은 `internal/lifecycle_test.go`와 `internal/watcher_*_test.go`에 있다.

## 안전 경계

- 웹훅 URL·토큰·비밀정보를 출력·오류·로그·커밋에 노출하지 않는다. 기존 URL 마스킹과 HTTP 클라이언트 로그 차단을 유지한다.
- 웹훅 검증에는 기존 테스트처럼 `httptest` 서버를 사용한다. 실제 Slack 전송 등 외부 상태 변경은 사용자의 명시적 요청이 있을 때만 수행한다.
- 사용자 요청 없이 커밋·푸시·릴리스를 실행하지 않는다. `go tool mage release`와 `release:*`는 README 변경, 커밋, 태그 생성과 원격 푸시를 수행하므로 검증 명령으로 실행하지 않는다.
