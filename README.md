# notion-site


[![](https://img.shields.io/github/v/release/nonacosa/notion-site.svg)](https://github.com/nonacosa/notion-site/releases)
[![](https://img.shields.io/github/license/nonacosa/notion-site.svg)](https://github.com/nonacosa/notion-site/blob/master/LICENSE)

**notion-site** is an open source software for a custom website based on [Notion](https://www.notion.so/) and [Hugo](https://gohugo.io/), and you can find your favorite template as your blog or documentation site among the hundreds of templates in the [Hugo Template Store](https://themes.gohugo.io/).

| Example                         | notion page |
|---------------------------------| --- |
| [doc](https://notion.zhuang.me) | [notion-page](https://zhuangwenda.notion.site/2bd00e5dfff3449ba81e0142f8af9bbb?v=065c41ad42be4683966e10f476e60afd) |
| [blog](https://blog-nonacosa.vercel.app/)    | [notion-page](https://zhuangwenda.notion.site/df7fb0e4e0114268b973f9d3e9a39982?v=557485cf3f564002acbdfd97c17ceb6f) |
| [website](https://findsofun.com/)    | [notion-page](https://aware-voyage-fde.notion.site/2527eff93d7e804ba921e3c44a094cc2?v=2527eff93d7e81ef9b71000c116cb7e4&pvs=73) |

 

![](img/notion-site.png)

## Requisites
- Notion Database id for your articles.
- Notion API secret token.


## Setup

### Unix system install

```bash
curl -sSf https://raw.githubusercontent.com/nonacosa/notion-site/master/install.sh | sh
```
 

## Debug local



```bash
cd your-hugo-site
notion-site init
# edit notion-site.yml your datatbse id and .env file notion secret
notion-site
```

### Github Action

> The installation command tool is helpful for local debugging. If you do not want to debug locally, you can also copy the configuration file to your project and run it directly through GitHubAction. You can see the example config in [notion-site-doc](https://github.com/nonacosa/notion-site-doc/blob/main/.github/workflows/builder.yml).

To use it as a Github Action, you can use the template  of the repository
in [.github/worflows/notion.yml](.github/workflows/notion.yml).

## Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details on submitting patches and the contribution workflow.

## Special thanks

- [go-notion](https://github.com/dstotijn/go-notion)
- [xzebra](https://github.com/xzebra)
- [saltbo](https://github.com/saltbo)


 
## License


notion-site is under the MIT license. See the [LICENSE](/LICENSE) file for details.
