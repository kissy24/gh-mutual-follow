# gh-mutual-follow CLI 仕様書

作成日: 2026-09-27

対象: GitHub CLI拡張として新規開発する `gh-mutual-follow`

状態: 初版の実装仕様案

## 1. 背景と目的

GitHubで相互フォローになっていない相手を確認し、フォローバックやフォロー解除を少ない手数で行うCLIを提供する。

既存のTUIは、起動、ペイン切り替え、対象選択という操作が必要になる。定期的に同じ整理をする用途では、目的に対応したコマンドを直接実行できる方が使いやすい。

本仕様は既存TUIの改修計画ではなく、一から開発するCLIの仕様とする。既存実装の構造やキーバインドとの互換性は要求しない。

### 方針と本書の位置づけ

- 会話で示された方針: CLIで提供する、TUIの操作負担をなくす、新規開発を前提にする。
- 本書で具体化した提案: GitHub CLI拡張での配布、以下のコマンド構成、フラグ、出力形式、エラー処理、初版の対応範囲。
- 本書の作成は、実装や公開を実施したことを意味しない。

## 2. 提供形態と対応範囲

| 項目 | 初版の仕様 |
| --- | --- |
| リポジトリ・実行ファイル名 | `gh-mutual-follow` |
| 利用時のコマンド | `gh mutual-follow` |
| 提供形態 | GitHub CLIのコンパイル済み拡張 |
| 実装言語 | Go |
| 認証 | GitHub CLIが利用する認証情報を使用 |
| 対象サービス | GitHub.com |
| 操作するアカウント | 実行時の認証ユーザー自身 |
| 初版の配布先 | macOS / Linuxのamd64・arm64 |
| 表示言語 | コマンドのヘルプ・出力は英語、READMEは日本語 |

GitHub CLIはGo製の拡張の雛形作成に対応する。配布の詳細は[公式の拡張作成ドキュメント](https://cli.github.com/manual/gh_extension_create)に従う。

### 初版に含める機能

- 片方向のフォロー関係の一覧。
- 指定ユーザー、または該当者全員へのフォローバック。
- 指定ユーザー、または該当者全員のフォロー解除。
- 対象からの除外、実行前確認、ドライラン。
- JSON出力と終了コードによるスクリプト連携。

### 初版に含めない機能

- TUI、対話的な一覧選択、ブラウザー画面。
- 常駐、スケジュール実行、自動フォロー監視。
- 相互フォロー相手の解除、任意の第三者への新規フォロー。
- 永続的な除外設定、操作履歴の保存、取り消し機能。
- GitHub Enterprise Server、Windows向け配布。
- 既存TUIとの互換モード。

## 3. 用語と対象判定

認証ユーザーのフォロー中ユーザー集合を `F`、フォロワー集合を `R` とする。

| 区分 | 定義 | CLI上の値 | 可能な操作 |
| --- | --- | --- | --- |
| 自分だけがフォロー中 | `F − R` | `following-only` | `unfollow` |
| 相手だけがフォロー中 | `R − F` | `followers-only` | `follow-back` |
| 相互フォロー | `F ∩ R` | 初版では出力対象外 | 操作対象外 |

- 関係データは毎回取得し、ローカルキャッシュを使わない。
- 全ページの取得が完了してから差分を計算する。取得途中のデータで変更を開始しない。
- 同一ユーザーの照合にはAPIのユーザーIDを用いる。CLI引数のログイン名は大文字小文字を区別せず照合する。
- 重複は除去し、ログイン名の大文字小文字を無視した昇順で表示・処理する。
- 取得結果に本人が含まれても対象から除く。
- 一覧取得はGitHub全体の同時点のスナップショットではない。取得中・確認待ち・実行中に他の操作で関係が変わる可能性は残る。

## 4. コマンド体系

```text
gh mutual-follow
gh mutual-follow list [--type TYPE] [--json]
gh mutual-follow follow-back [USER...] [--all] [--exclude USER]... [--dry-run] [--yes] [--json]
gh mutual-follow unfollow [USER...] [--all] [--exclude USER]... [--dry-run] [--yes] [--json]
gh mutual-follow --help
gh mutual-follow --version
```

- サブコマンドなしではヘルプと代表例を表示し、終了コード `0` で終了する。APIを呼ばない。
- `--help` は各サブコマンドでも利用できる。`--version` とともに認証不要とする。
- 不明なコマンド、不明なフラグ、不正な組み合わせはAPI呼び出し前に拒否する。
- ローカルのGitリポジトリに依存せず、任意のディレクトリで実行できる。

### 4.1 list

```sh
# 片方向の関係を両方表示
gh mutual-follow list

# 自分だけがフォロー中の相手
gh mutual-follow list --type following-only

# まだフォローバックしていない相手
gh mutual-follow list --type followers-only

# スクリプトで利用
gh mutual-follow list --json
```

`--type` は `all`、`following-only`、`followers-only` を受け付け、既定値は `all` とする。ここで `all` は片方向の2区分を意味する。

通常出力はアカウント、区分別件数、ログイン名、プロフィールURLを表示する。該当者がいなければ、その旨を表示して正常終了する。全件を出力し、30人などの表示上限や自動ページャーは設けない。

```text
Account: octocat (github.com)

Following only: 2
alice    https://github.com/alice
charlie  https://github.com/charlie

Followers only: 1
bob      https://github.com/bob
```

### 4.2 follow-back

```sh
# 特定の相手をフォローバック
gh mutual-follow follow-back bob

# 対象全員をフォローバック
gh mutual-follow follow-back --all

# 一部を除外し、確認を省略
gh mutual-follow follow-back --all --exclude bob --yes
```

操作可能なのは `followers-only` に含まれる相手だけとする。

### 4.3 unfollow

```sh
# 特定の相手を解除
gh mutual-follow unfollow alice charlie

# 全対象の確認のみ
gh mutual-follow unfollow --all --dry-run

# フォローを続けたい相手を除いて解除
gh mutual-follow unfollow --all --exclude alice
```

操作可能なのは `following-only` に含まれる相手だけとする。相互フォロー中のユーザーは、名前を明示されても解除しない。

### 4.4 変更コマンドの共通フラグ

| フラグ | 動作 |
| --- | --- |
| `--all` | 該当区分の全ユーザーを選ぶ。現在のページなどの概念はない |
| `--exclude USER` | 選択対象から除外する。繰り返し指定可能。カンマ区切りは扱わない |
| `--dry-run` | 最新データから対象と除外結果を表示する。変更APIと確認入力を実行しない |
| `--yes`, `-y` | 確認入力を省略する。対象判定・除外・エラー検出は省略しない |
| `--json` | 最終結果を1つのJSONオブジェクトで出力する |

対象指定の規則:

1. `USER...` と `--all` はどちらか一方を必須とする。両方指定、両方省略は使用方法エラー。
2. 明示されたログイン名の重複はまとめる。
3. 明示されたユーザーが取得済みの該当区分にいなければ、全体を検証エラーとして扱い、1件も変更しない。未存在、相互フォロー、すでに処理済みの場合も同じ扱いとする。
4. 対象の検証後、`--exclude` を適用する。対象外の除外指定は無視する。
5. 除外後の対象が0人なら、確認を求めず「変更なし」として正常終了する。
6. `--dry-run --yes` は許可するが、ドライランを優先し変更しない。
7. `--json` による変更実行には `--yes` を必須とする。`--dry-run --json` では不要。

## 5. 確認と実行の流れ

変更コマンドは、引数検証 → 認証ユーザー取得 → 全件取得 → 対象判定 → 除外 → 確認 → 順次実行 → 結果出力の順に進む。

通常の対話実行では、アカウント、操作内容、対象者全員、対象件数、除外件数を標準エラー出力に表示し、1回だけ確認する。

```text
Account: octocat (github.com)
Action: unfollow
Targets: 2 / Excluded: 1
  charlie
  dave
Unfollow these 2 users? [y/N]:
```

- 大文字小文字を無視した `y` / `yes` だけを承認とする。その他の入力、空入力、EOFは中止とする。
- 標準入力が端末でない場合、`--yes` がなければ入力待ちせず終了する。ただしドライランと対象0人の場合は正常終了する。
- `--yes` 使用時は確認画面を省略し、最終結果にアカウントと処理内容を含める。
- 確認した対象集合を固定する。確認後の再取得で対象を自動追加しない。
- APIによる変更は直列に実行し、処理中に別の変更処理を並列起動しない。
- 1件のリクエストには30秒のタイムアウトを設ける。初版では自動再試行を行わない。
- 最初の変更失敗で後続処理を停止する。成功済みの変更は巻き戻さず、未着手分を区別して報告する。
- Ctrl+Cでは後続処理を開始せず、実行中のリクエストをキャンセルする。取得できた処理結果を出力する。
- 通信切断・タイムアウト・キャンセルによりサーバー側の反映が判断できない対象は「結果不明」とする。

ドライランは、その実行時点での予定を示す。後日同じコマンドを実行したときの対象一致は保証しない。実行後に最新の関係を確認したい場合は `list` を再実行する。

## 6. 出力と終了コード

### 通常出力

- 標準出力: 一覧、ドライラン結果、変更結果。
- 標準エラー出力: 確認、端末向け進捗、エラーの補足。
- 変更結果にはアカウント、操作、対象ごとの結果、集計を含める。
- 進捗は端末接続時だけ表示し、画面全体の再描画やキー操作待ちは行わない。
- ドライランでは `Would follow` / `Would unfollow` と明示し、成功件数として数えない。

```text
Account: octocat (github.com)
Action: unfollow
SUCCESS  charlie
FAILED   dave      Permission denied
NOT_RUN  eve
Targets: 3 / Succeeded: 1 / Failed: 1 / Unknown: 0 / Not run: 1 / Excluded: 1
```

### JSON出力

`--json` では標準出力にJSON以外を混ぜず、端末向けの進捗も表示しない。空配列は `[]` とし、通常のエラーや中止でも可能な限り1つの結果オブジェクトを返す。強制終了時の出力は保証しない。

共通フィールド:

| フィールド | 型・意味 |
| --- | --- |
| `schema_version` | 整数。初版は `1` |
| `command` | `list` / `follow-back` / `unfollow` |
| `account` | 認証ユーザーのログイン名。取得前の失敗では `null` |
| `host` | `github.com` |
| `dry_run` | 真偽値。`list` では `false` |
| `status` | `ok` / `error` / `cancelled` |
| `error` | `null` または `{ "code": "...", "message": "..." }` |

`list` は `users` 配列を追加する。要素は `login`、`url`、`relation` を持ち、`relation` は第3節の区分値とする。`--type` 適用後の配列を返す。

変更コマンドは `results` と `summary` を追加する。

- `results` の各要素: `login`、`status`、`error`。対象者と除外者を含め、ログイン名順とする。
- 対象ごとの `status`: `planned` / `succeeded` / `failed` / `unknown` / `not_run` / `excluded`。
- `summary`: `targets`、`planned`、`succeeded`、`failed`、`unknown`、`not_run`、`excluded` の整数値。
- `targets` は除外後の人数で、`planned + succeeded + failed + unknown + not_run` に一致する。
- ドライランは全対象を `planned` とする。確認中止は全対象を `not_run` とする。
- 対象確定前のエラーは空配列・全件数0とし、`error` で理由を示す。
- 主なエラーコード: `invalid_arguments`、`invalid_target`、`authentication_failed`、`permission_denied`、`rate_limited`、`api_error`、`outcome_unknown`、`confirmation_required`、`cancelled`。
- 初版中はフィールドの意味を変更しない。追加フィールドを利用者が無視できる設計とする。

### 終了コード

以下は本ツール独自の定義であり、内部で起動した `gh` の終了コードをそのまま返さない。

| コード | 意味 |
| --- | --- |
| `0` | 成功。ヘルプ、ドライラン、対象0人を含む |
| `1` | 認証・取得・変更などの実行失敗。部分成功、結果不明を含む |
| `2` | 引数・対象の検証エラー、非対話実行に必要な `--yes` の不足 |
| `3` | 確認入力による中止 |
| `130` | Ctrl+Cによる中断 |

## 7. GitHub連携

### 認証とホスト

- `gh` が利用する認証情報を再利用し、独自のログイン処理・トークン保存機能は作らない。
- 認証ユーザーは `GET /user` のJSONで判定する。`gh auth status` の表示文字列を解析しない。
- 全APIリクエストのホストを `github.com` に明示固定し、カレントディレクトリのリモート設定で変化させない。
- 未認証時は `gh auth login --hostname github.com` を案内する。
- OAuth・classic PATによる変更には `user:follow` が必要。fine-grained PATでは、認証ユーザーの一覧取得にFollowersのread、変更にwrite権限が必要となる。権限不足時は認証方式に応じた案内を表示する。[GitHub公式API仕様](https://docs.github.com/en/rest/users/followers)
- トークンの値は出力・保存しない。認証の切り替えは利用者が `gh` 側で行う。

### APIとページネーション

| 用途 | メソッド・パス |
| --- | --- |
| 認証ユーザー | `GET /user` |
| フォロー中の一覧 | `GET /user/following` |
| フォロワー一覧 | `GET /user/followers` |
| フォロー | `PUT /user/following/{username}` |
| フォロー解除 | `DELETE /user/following/{username}` |

フォロー関係のエンドポイントと権限は[GitHub公式API仕様](https://docs.github.com/en/rest/users/followers)を参照する。

一覧は1ページ100件を指定し、次ページがなくなるまで取得する。`gh api --paginate` を使う場合、ページごとに独立したJSONが返るため、JSONストリームとして順次デコードするか、`--slurp` の配列を平坦化する。単一のユーザー配列として誤って解析しない。[gh api公式仕様](https://cli.github.com/manual/gh_api)

認証エラー、権限不足、レート制限、通信失敗を区別して案内する。レート制限時は後続の変更を停止し、応答から分かる範囲で再実行の目安を表示する。失敗や結果不明を成功扱いしない。

## 8. 実装方針

- CLI引数解析、対象計算、操作実行、GitHub通信、出力を分離する。
- 集合差分と対象選択は、通信を含まない関数としてテスト可能にする。
- GitHub通信をインターフェース化し、通常のテストで実アカウントを書き換えない。
- GitHub連携には `gh api` またはGitHub CLI向けGoライブラリを採用できる。HTTP状態・キャンセル・タイムアウトを扱えることを条件とする。
- 外部コマンドを使う場合はシェル文字列を組み立てず、引数配列を渡す。
- TUIライブラリは導入しない。CLI解析ライブラリの採用は実装時に決定する。
- TDDで、対象計算、コマンド契約、異常系から小さく実装する。

## 9. インストールと配布

公開後、利用者は以下の手順で導入できることを目標とする。`OWNER` は新規リポジトリの所有者に置き換える。

```sh
gh auth login --hostname github.com
gh extension install OWNER/gh-mutual-follow
gh mutual-follow list

# 更新・削除
gh extension upgrade mutual-follow
gh extension remove mutual-follow
```

開発時は、リポジトリのルートに同名の実行ファイルをビルドし、ローカル拡張として登録する。

```sh
go build -o gh-mutual-follow .
gh extension install .
gh mutual-follow --help
```

ローカルのコンパイル済み拡張では手動ビルドが必要となる。[公式インストール仕様](https://cli.github.com/manual/gh_extension_install)

- バージョンタグを契機にCIで対応OS・CPUのバイナリを生成し、GitHub Releasesに添付する。
- 拡張のリリース形式は公式Go拡張の雛形に合わせ、実際の `gh extension install` と `upgrade` で検証する。
- バイナリにはバージョン情報を埋め込む。
- READMEに導入、認証・権限、代表例、終了コード、ドライラン、除外方法、対応環境を記載する。
- 対応するGo・GitHub CLIの最低バージョンは実装時のCI検証結果から決定し、初回公開前にREADMEへ明記する。

## 10. 受け入れ条件

| 番号 | シナリオ | 期待結果 |
| --- | --- | --- |
| 1 | サブコマンドなし、ヘルプ、バージョン | API呼び出しなしで正常終了 |
| 2 | Followingがalice・bob、Followersがbob・charlie | following-onlyはalice、followers-onlyはcharlie |
| 3 | 100件を超える一覧 | 全ページを含む正しい差分を返す |
| 4 | 途中ページの取得に失敗 | 変更APIを1件も呼ばず失敗終了 |
| 5 | `unfollow --all --exclude alice --dry-run` | aliceを除く予定だけを表示し、変更しない |
| 6 | 明示対象に相互フォロー相手を含む | 対象全体を拒否し、変更しない |
| 7 | `USER...` と `--all` を併用・両方省略 | 使用方法エラー。変更しない |
| 8 | 確認で空入力・否認・EOF | 変更せず終了コード3 |
| 9 | 非端末入力で `--yes` なし | 入力待ちせず終了コード2。ただしドライラン・対象0人は0 |
| 10 | `--yes` による実行 | 確認待ちせず、対象だけに変更を実行 |
| 11 | 全対象を除外、または候補0人 | 変更・確認なしで正常終了 |
| 12 | 2人目の変更が明確に失敗 | 1人目は成功、2人目は失敗、残りは未着手として終了コード1 |
| 13 | 変更中に応答が途絶える | 結果不明を記録し、後続を停止。自動再試行しない |
| 14 | Ctrl+Cで中断 | 後続を開始せず、成功済み・結果不明・未着手を区別して終了コード130 |
| 15 | JSON出力 | 有効なJSONだけが標準出力に出て、集計と個別結果が一致 |
| 16 | `--dry-run --yes` | 変更しない |
| 17 | `--json` の変更で `--yes` を省略 | 確認を始めず、使用方法エラー |
| 18 | 大文字小文字違い・重複した名前 | 同一ユーザーを二重処理しない |
| 19 | 未認証・権限不足・レート制限 | 理由と対処を表示し、正常終了にしない |
| 20 | Gitリポジトリ外で起動 | 同じアカウント・ホストで動作 |

APIモックを用いた自動テストを中心とし、実アカウントの変更を伴う検証は専用アカウントで明示的に行う。配布確認では対応環境ごとにインストール、ヘルプ表示、読み取りコマンド、更新を検証する。

## 11. 実装順序

1. CLIの入口、ヘルプ、バージョン、終了コード。
2. 認証ユーザー取得、全ページ取得、差分計算、`list`。
3. 対象選択、除外、ドライラン、確認入力。
4. `follow-back` / `unfollow`、部分失敗、中断処理。
5. JSON出力とコマンド単位の受け入れテスト。
6. README、リリースCI、拡張のインストール・更新検証。

初版の完成条件は、第10節を満たし、利用者が導入から一覧確認・ドライラン・実行までREADMEだけで進められることとする。
