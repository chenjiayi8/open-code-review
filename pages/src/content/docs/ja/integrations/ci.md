---
title: CI/CD
sidebar:
  order: 4
---

すべての Pull Request または Merge Request で OCR を実行します。上流リポジトリは、コピーして設定するだけのすぐ使える 2 つのパイプラインを提供しています——1 つは GitHub Actions、もう 1 つは GitLab CI です。どちらも
[CLI リファレンス](../cli-reference/#json)に記載されている中核コマンドの薄いラッパーです。

## CI/CD 統合の仕組み

本ページの各レシピは同じパターンに従います——以下の GitHub Actions と GitLab CI のセクションは、その具体的な実装に過ぎません。

1. **PR / MR イベントでトリガー。** 新しい pull request、更新された merge request、または手動の
   `/open-code-review` コメントがジョブをトリガーします。
2. **runner に `ocr` をインストール**します。通常は
   `npm install -g @alibaba-group/open-code-review` です。runner は一時的なため、これは実行のたびに発生します。
3. **CI job 内で選択したローカル runner を認証**します。Codex または Claude をインストールし、CI プラットフォームがサポートする非対話認証（承認済み secret、OIDC exchange、device-flow/bootstrap 手順など）を使います。OCR 自体はモデルサービス認証情報を直接消費しません。
4. **区間モードでレビューを実行**し、機械可読な出力を得ることで、stdout がクリーンな JSON の外殻になるようにします。

   ```bash
   ocr review --runner codex \
     --from "origin/<base-branch>" \
     --to "origin/<head-branch>" \
     --format json \
     --audience agent
   ```

   `--format json` は解析可能なペイロードを提供し、`--audience agent` は進捗行を抑制します。各レシピが消費する外殻は [JSON 出力](../cli-reference/#json)を参照してください。
5. **JSON を解析**し、`comments[]` を反復処理します。
6. **プロバイダーの review API を通じてコメントを PR / MR に貼り戻します。** 有効な行情報を持たない項目（ファイルレベルの発見）はインラインで貼り付けるのではなくサマリーの注記にまとめられます。インラインの一括 API がリクエストを拒否した場合、貼り付け手順も通常のサマリーコメントにフォールバックします。

常に 2 種類の認証情報が関わります。Codex または Claude が発見を生成するために使う、選択された **runner 認証**と、貼り付け手順がコメントを貼り戻すために使う **PR/MR 書き込み token** です。GitHub のレシピは `GITHUB_TOKEN` を通じて後者を自動的に提供します。GitLab では `GITLAB_API_TOKEN` を明示的に設定することを推奨しますが、fork MR に対しては組み込みの `CI_JOB_TOKEN` にフォールバックします（これは `/discussions` を通じてディスカッションを開始できます）——信頼性のためには専用の token の使用を推奨します。

## GitHub Actions

上流のワークフローは
[`examples/github_actions/ocr-review.yml`](https://github.com/alibaba/open-code-review/blob/main/examples/github_actions/ocr-review.yml)
にあります。

### 何をするか

- `pull_request_target`（`opened`）**および** `issue_comment` イベント（本文が
  `/open-code-review` または `@open-code-review` で始まるもの）でトリガーします。後者はレビュアーが PR にコメントすることで OCR をオンデマンドで再実行できるようにします。（`pull_request` ではなく `pull_request_target` を使うことで、fork から提出された PR でも secret を利用できます。OCR は diff を読むだけで、PR 内のコードは実行しません。）
- `npm install -g @alibaba-group/open-code-review` で OCR をインストールし、選択した Codex または Claude runner を認証し、ブランチ区間モードで中核コマンドを実行します。
- JSON の外殻を解析し、GitHub Pull Request Review API を通じて各発見をインラインのレビューコメントとして貼り付けます。行情報を持たないコメントはサマリー本文にまとめられます。一括送信が失敗した場合は 1 件ずつの貼り付けにフォールバックし、統計をサマリーコメントに表示します。

### インストール

ワークフローをリポジトリに配置します。

```bash
mkdir -p .github/workflows
curl -o .github/workflows/ocr-review.yml \
  https://raw.githubusercontent.com/alibaba/open-code-review/main/examples/github_actions/ocr-review.yml
```

### 必須の secret

**Settings → Secrets and variables → Actions** で設定します。

| Secret | 必須 | 説明 |
|---|---|---|
| `GITHUB_TOKEN` | 自動 | PR review コメント投稿用に GitHub が提供します。 |

OCR の前に、選択したインストール済み runner の公式にサポートされた CI 方式で認証するステップを追加し、`--runner codex` または `--runner claude` で OCR を実行します。OCR 独自の runner secret 名は作成しません。
### カスタマイズ

以下はすべて、あなたがコピーしたばかりのワークフローファイル
（`.github/workflows/ocr-review.yml`）への編集です。

#### 背景コンテキスト

`--background` は最も効果の大きい単一の引数です——[すべてのモードに適用されるヒント](../#tips-that-apply-to-every-pattern)を参照してください。PR タイトルを渡します（タイトルが `feat(auth): add OAuth2 support` のようなセマンティックな規約に従っている場合、より効果的です）。

```yaml
- name: Run OCR review
  env:
    PR_TITLE: ${{ github.event.pull_request.title }}
    BASE_REF: ${{ github.base_ref }}
    HEAD_REF: ${{ github.head_ref }}
  run: |
    ocr review --runner codex \
      --background "$PR_TITLE" \
      --from "origin/$BASE_REF" \
      --to "origin/$HEAD_REF" \
      --format json --audience agent
```

PR で制御可能な値は `${{ }}` を `run:` に直接展開するのではなく、`env:` 経由で渡してください。GitHub は `${{ }}` を shell が行を解析する *前に* テキストとして置換するため、shell のメタ文字を含む PR タイトルやブランチ名が runner 上で実行されてしまいます。

#### カスタムルール

`--rule` でプロジェクト固有のルールファイルを渡します。

```yaml
- name: Run OCR review
  env:
    BASE_REF: ${{ github.base_ref }}
    HEAD_REF: ${{ github.head_ref }}
  run: |
    ocr review --runner codex --rule ./my-rules.json \
      --from "origin/$BASE_REF" \
      --to "origin/$HEAD_REF"
```

スキーマは[レビュールール](../../review-rules/)を参照してください。

#### Runner timeout

`--timeout <minutes>` は local runner process 全体を制限します。大きな PR では、review range、`--exclude`、または path-limited scan で selection を狭めて、選択した runner の quota 消費を抑えてください。

```yaml
- name: Run OCR review
  env:
    BASE_REF: ${{ github.base_ref }}
    HEAD_REF: ${{ github.head_ref }}
  run: |
    ocr review --runner codex \
      --from "origin/$BASE_REF" \
      --to "origin/$HEAD_REF"
```

#### トリガーモード

デフォルトのワークフローは、PR が **opened** されたとき、および `/open-code-review` または
`@open-code-review` で始まる PR コメントのときにトリガーします。よくある 2 つの調整があります。

より多くの PR ライフサイクルイベントで実行する（例：新しい commit がプッシュされたときに再レビュー）：

```yaml
on:
  pull_request:
    types: [opened, synchronize, reopened, ready_for_review]
```

異なるコメントキーワードを使う：

```yaml
if: |
  github.event_name == 'pull_request' ||
  (github.event_name == 'issue_comment'
    && github.event.issue.pull_request
    && startsWith(github.event.comment.body, '/review'))
```

`github.event.issue.pull_request` のチェックは、コメントが通常の issue ではなく PR 上のものであることを保証します。

#### OCR のバージョン固定

デフォルトのワークフローは最新のリリース版をインストールします。固定するには：

```yaml
- name: Install OpenCodeReview
  run: npm install -g @alibaba-group/open-code-review@1.0.0
```

#### GitHub App として投稿する

デフォルトではレビューコメントは `github-actions[bot]` から投稿されます。`OpenCodeReview Bot` のようなカスタムブランドの bot として投稿するには、`GITHUB_TOKEN` を GitHub App の installation token に置き換えます。

1. *Settings → Developer settings → GitHub Apps → New GitHub App* で **app を作成**します。webhook は無効にします（このユースケースでは不要）。*Repository permissions* で次を付与します。
   - **Pull requests**：Read and write
   - **Contents**：Read-only（diff の取得用）
   - **Metadata**：Read-only（必須）

2. app 設定ページから**秘密鍵を生成**し、`.pem` ファイルをダウンロードします。同じページの **App ID** を控えておきます。

3. app を OCR にレビューさせたいリポジトリに**インストール**します。Installation ID はインストール後の URL に現れます。例：`https://github.com/settings/installations/12345` → ID は `12345`。

4. *Settings → Secrets and variables → Actions* で**3 つの secret を追加**します。

   | Secret | 値 |
   |---|---|
   | `GITHUB_APP_ID` | App ID。 |
   | `GITHUB_APP_PRIVATE_KEY` | `.pem` ファイルの全内容。`-----BEGIN RSA PRIVATE KEY-----` と `-----END RSA PRIVATE KEY-----` の行を含みます。 |
   | `GITHUB_APP_INSTALLATION_ID` | Installation ID。 |

5. コメント貼り付け手順で **token を生成して使用**します。

   ```yaml
   - name: Get GitHub App Token
     id: app-token
     uses: actions/create-github-app-token@v1
     with:
       app-id: ${{ secrets.GITHUB_APP_ID }}
       private-key: ${{ secrets.GITHUB_APP_PRIVATE_KEY }}

   - name: Post review comments to PR
     uses: actions/github-script@v7
     with:
       github-token: ${{ steps.app-token.outputs.token }}
       script: |
         # ...existing post script...
   ```

レビューは `github-actions[bot]` ではなく、あなたの app の名前で投稿されるようになります。

### トラブルシューティング

| 症状 | 原因 / 修正 |
|---|---|
| `Cannot find merge-base` | checkout 手順が浅いクローンを使っていますが、区間モードのレビューには完全な履歴が必要です。上流のワークフローは `actions/checkout` に `fetch-depth: 0` を設定しています——ファイルを編集する際はこの設定を保持してください。 |
| `Failed to parse OCR output` | 選択した runner が CI で認証されていません。Codex/Claude のログイン手順を確認して job を再実行してください。 |
| レビューコメントが誤った行に付く | 通常、レビュー開始からコメント貼り付けの間に diff がずれたことを意味します。貼り付けスクリプトはこの場合、通常の issue コメントにフォールバックします——対処は不要です。 |

> **手軽な bot 命名のヒント。** Project Access Token と Group Access Token では、
## GitLab CI

上流のパイプラインは
[`examples/gitlab_ci/.gitlab-ci.yml`](https://github.com/alibaba/open-code-review/blob/main/examples/gitlab_ci/.gitlab-ci.yml) にあります。

### 何をするか

- `merge_requests` イベントでトリガーします。
- Node イメージで OCR をインストールし、選択した Codex または Claude runner を認証してから、`--runner` 付きで MR diff モードのコアコマンドを実行します。
- JSON エンベロープを解析し、各発見を GitLab Discussion として投稿します。インライン位置を特定できない場合は通常の MR note にフォールバックします。

### インストール

パイプラインをリポジトリルートに配置します。

```bash
curl -o .gitlab-ci.yml \
  https://raw.githubusercontent.com/alibaba/open-code-review/main/examples/gitlab_ci/.gitlab-ci.yml
```

既存の `.gitlab-ci.yml` がある場合は、別パスに vendoring して include できます。

```yaml
include:
  - local: 'ci/ocr-review.gitlab-ci.yml'
```

### 必須 CI/CD 変数

**Settings → CI/CD → Variables** で設定します。

| 変数 | 必須 | Masked | 説明 |
|---|---|---|---|
| `GITLAB_API_TOKEN` | いいえ | はい | コメント投稿用の project / personal / group access token。scope は `api`。任意です。fork MR では組み込みの `CI_JOB_TOKEN` にフォールバックできます。 |

OCR の前に、選択したインストール済み runner の公式にサポートされた CI 方式で認証するパイプライン手順を追加し、`--runner codex` または `--runner claude` で OCR を呼び出します。OCR 独自の runner secret 名は作成しません。

> token の**名前**が MR ディスカッションの横に表示されます。token を `OpenCodeReview Bot` と命名すれば、追加設定なしでレビューディスカッションにブランド名を付けられます——[サービスアカウント名義で投稿する](#post-under-a-service-account-identity)に記載のより永続的なサービスアカウント設定が不要なときに便利です。

### カスタマイズ

以下はすべて、あなたがコピーしたばかりの `.gitlab-ci.yml` への編集です。

#### 背景コンテキスト

MR タイトルを `--background` に渡します——タイトルが `feat(auth): add OAuth2 support`
のようなセマンティックな規約に従っている場合、より効果的です。

```yaml
script:
  - |
    ocr review --runner codex \
      --background "$CI_MERGE_REQUEST_TITLE" \
      --from "origin/$CI_MERGE_REQUEST_TARGET_BRANCH_NAME" \
      --to "${CI_COMMIT_SHA}" \
      --format json --audience agent
```

#### カスタムルールと runner timeout

GitHub Actions のレシピと同じ引数です——`--rule` でプロジェクト固有のルールファイルを渡し、
`--timeout` bounds the overall local runner process:

```yaml
script:
  - |
    ocr review --runner codex --rule ./my-rules.json \
      --from "origin/$CI_MERGE_REQUEST_TARGET_BRANCH_NAME" \
      --to "${CI_COMMIT_SHA}"
```

ルールのスキーマは[レビュールール](../../review-rules/)を参照してください。

#### OCR のバージョン固定

```yaml
script:
  - npm install -g @alibaba-group/open-code-review@1.0.0
```

#### プッシュのたびの再レビューを避ける

`only: [merge_requests]` は **MR の更新のたびに**トリガーするため、長期にわたる MR ではrunner の quota を大量に消費します。GitLab にはネイティブの「作成時のみ」イベントがないため、推奨されるパターンは、レビューを実行する前に既存の OCR note を検出し、あればスキップすることです。`ocr review --runner codex` の呼び出しを Python wrapper に置き換えます。

```python
import json, os, sys, urllib.request

GITLAB_URL = os.environ.get("CI_SERVER_URL", "https://gitlab.com")
PROJECT_ID = os.environ["CI_PROJECT_ID"]
MR_IID     = os.environ["CI_MERGE_REQUEST_IID"]
API_TOKEN  = os.environ["GITLAB_API_TOKEN"]

url = (
    f"{GITLAB_URL}/api/v4/projects/{PROJECT_ID}"
    f"/merge_requests/{MR_IID}/notes?per_page=100"
)
req = urllib.request.Request(url, headers={"PRIVATE-TOKEN": API_TOKEN})
with urllib.request.urlopen(req) as resp:
    notes = json.loads(resp.read().decode())

if any("OpenCodeReview" in n.get("body", "") for n in notes):
    print("OCR already reviewed this MR. Skipping to save tokens.")
    sys.exit(0)

# ...otherwise call `ocr review --runner codex ...` as usual and write the JSON to
# the file the posting step expects.
```

この後に再レビューを強制するには、MR から以前の OCR note を削除してください——次のパイプライン実行では OCR note が見当たらなくなり、処理を続行します。

#### セルフホスト GitLab

コードの変更は不要です。貼り付けスクリプトは `CI_SERVER_URL`（GitLab が各 runner で自動的に設定します）を読むため、そのままで自分のインスタンスと通信できます。`GITLAB_API_TOKEN` が `gitlab.com` ではなく、あなたのセルフホストインスタンスによって発行されていることだけ確認してください。

#### サービスアカウント名義で投稿する

デフォルトではレビューディスカッションは `GITLAB_API_TOKEN` が属するユーザー名で表示されます。プロジェクトレベルのサービスアカウントに切り替えると、`OpenCodeReview Bot` のようなカスタムブランドの bot 名義が得られます。

1. *Project → Settings → Service Accounts → New service account* で**サービスアカウントを作成**します。選んだ名前（例：`OpenCodeReview Bot`）が MR ディスカッションの横に表示されます。

2. *Settings → Members → Invite member* で**プロジェクトに招待**します。サービスアカウント名を検索し、`Developer` または `Maintainer` を割り当てます——どちらもディスカッションの投稿に必要な権限を持ちます。

3. *Settings → Service Accounts →（該当アカウント）→ Add new token* で **access token を発行**します。必要な scope は `api` です。token はすぐにコピーしてください——GitLab は一度しか表示しません。

4. *Settings → CI/CD → Variables* で **token の値を置き換え**ます——既存の `GITLAB_API_TOKEN` の値をサービスアカウントの token で置き換えます（変数名は変えません）。

ディスカッションは、最初に token を作成したユーザー名ではなく、サービスアカウント名で投稿されるようになります。

### トラブルシューティング

| 症状 | 原因 / 修正 |
|---|---|
| `Cannot find merge-base` | runner が浅いクローンを使っています。上流のパイプラインは `GIT_DEPTH: 0` を設定して完全なクローンを強制します——ファイルを編集する際はこの設定を保持してください。 |
| 投稿時の `API error 403` | `GITLAB_API_TOKEN` に `api` scope が無い、プロジェクトのメンバーでない、または——セルフホストの場合——別のインスタンスによって発行されています。`api` scope で再発行し、*Settings → CI/CD → Variables* で再登録してください。 |
| `Failed to parse OCR output` | 選択した runner が CI で認証されていません。Codex/Claude のログイン手順を確認して job を再実行してください。 |
| インラインコメントが誤った行に付く | GitLab のインラインディスカッションは正確な SHA の一致を要求します。貼り付けスクリプトは `versions` メタデータを取得して正しい `base_sha` / `start_sha` / `head_sha` を得ます。それでも発見をアンカーできない場合は、通常の MR note にフォールバックします。 |

パイプラインは生のレビュー JSON を `/tmp/ocr-result.json` に、stderr を
`/tmp/ocr-stderr.log` に書き込みます。debug 手順でそれらを cat して、OCR が何を返したか確認できます。

```yaml
script:
  - cat /tmp/ocr-result.json
  - cat /tmp/ocr-stderr.log
```

## 関連項目

- [CLI リファレンス](../cli-reference/#json)——2 つのパイプラインが消費する JSON の構造。ゼロから CI スクリプトを書くときに役立ちます。
- [設定](../../configuration/)——OCR が受け付けるすべての環境変数と config key。
