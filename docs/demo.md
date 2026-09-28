# 副作用のないCLIデモ

実施日: 2026-09-28

実装済みのCLI処理を、架空データのクライアントで実行した記録。GitHubへの通信、認証情報の読み取り、フォロー関係の変更は行っていない。

自分だけがフォロー中: alice・charlie。相手だけがフォロー中: dave・eve。相互フォロー: bob。

以下のコマンド表記は実際の利用時の呼び出しに対応する。今回の実行では一時的なGoプログラムから同じCLI処理を呼び出し、通常のGitHubクライアントを使用していない。

## 1. 一覧を表示

```text
$ gh mutual-follow list
Account: demo-user (github.com)

following-only: 2
alice	https://github.com/alice
charlie	https://github.com/charlie

followers-only: 2
dave	https://github.com/dave
eve	https://github.com/eve
[exit code: 0]
```

## 2. フォローバックの予定を確認

```text
$ gh mutual-follow follow-back --all --dry-run
Account: demo-user (github.com)
Action: follow-back
Would follow	dave
Would follow	eve
Targets: 2 / Planned: 2 / Succeeded: 0 / Failed: 0 / Unknown: 0 / Not run: 0 / Excluded: 0
[exit code: 0]
```

## 3. aliceを残して解除予定を確認

```text
$ gh mutual-follow unfollow --all --exclude alice --dry-run
Account: demo-user (github.com)
Action: unfollow
EXCLUDED	alice
Would unfollow	charlie
Targets: 1 / Planned: 1 / Succeeded: 0 / Failed: 0 / Unknown: 0 / Not run: 0 / Excluded: 1
[exit code: 0]
```

## 4. 実行前の確認で中止

```text
$ gh mutual-follow unfollow --all
Account: demo-user (github.com)
Action: unfollow
Targets: 2 / Excluded: 0
  alice
  charlie
Apply unfollow to these 2 users? [y/N]: n
Account: demo-user (github.com)
Action: unfollow
NOT_RUN	alice
NOT_RUN	charlie
Targets: 2 / Planned: 0 / Succeeded: 0 / Failed: 0 / Unknown: 0 / Not run: 2 / Excluded: 0
Cancelled; no changes made.
[exit code: 3]
```

## 5. 相互フォロー相手の解除を拒否

```text
$ gh mutual-follow unfollow bob --dry-run
Account: demo-user (github.com)
Action: unfollow
Targets: 0 / Planned: 0 / Succeeded: 0 / Failed: 0 / Unknown: 0 / Not run: 0 / Excluded: 0
bob is not a following-only target; no changes made
[exit code: 2]
```

変更メソッドの呼び出し: 0回。デモ用プログラムは実行後に削除。製品コード・仕様の変更なし。
