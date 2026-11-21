# Third-Party Notices

kdash 包含下列第三方字体和 Go 模块。kdash 自身的 MIT 许可证不替代这些组件各自的许可证。分发包含这些组件的 kdash 时，应同时分发本文件、`assets/fonts/licenses/` 中相应的 OFL 文本，以及字体二进制中原有的版权和许可元数据。

## 1. Bundled fonts

### 核实方法和范围

- 核对了全部 11 个 TTF 的 `name` 表：版权（ID 0）、唯一标识（ID 3）、完整字体名（ID 4）、版本（ID 5）、许可说明（ID 13）和许可 URL（ID 14）。
- 下列“二进制版权”逐字取自本地字体；“OFL 版权”取自对应官方上游 OFL 文件。两者不同时均予保留，不以后者覆盖前者。
- 所有字体均使用 **SIL Open Font License 1.1**。每个字体族都有独立的完整 OFL 文本，路径见各条目。
- Bebas Neue、Long Cang、Ma Shan Zheng、ZCOOL XiaoWei、ZCOOL KuaiLe、JetBrains Mono 以及三个 Noto Sans SC 二进制已与下列官方 URL 做 SHA-256 比对并一致。
- Montserrat Thin 和 Playfair Display Regular 的家族、版本、二进制版权元数据及官方 OFL 已确认；但本地文件与当前上游静态构建并非字节一致，未据此声称找到了完全相同的发布构建。其本地 SHA-256 仍列出以便识别实际分发文件。

### Bebas Neue — `BebasNeue.ttf`

- 实际字体/版本：Bebas Neue Regular，Version 2.000。
- 二进制版权：`Copyright 2019 The Bebas Neue Project Authors (https://github.com/dharmatype/Bebas-Neue)`。
- OFL 版权：`Copyright © 2010 by Dharma Type.`
- 官方项目：<https://github.com/dharmatype/Bebas-Neue>。
- 官方分发及哈希比对来源：<https://raw.githubusercontent.com/google/fonts/main/ofl/bebasneue/BebasNeue-Regular.ttf>。
- 本地 SHA-256：`08e4623805102d819f58601e46e345648846075e363b2ceb23313c2d1c83ec73`。
- 许可证文件：[`assets/fonts/licenses/BebasNeue-OFL.txt`](assets/fonts/licenses/BebasNeue-OFL.txt)；上游原文：<https://raw.githubusercontent.com/google/fonts/main/ofl/bebasneue/OFL.txt>（与 <https://raw.githubusercontent.com/dharmatype/Bebas-Neue/master/OFL.txt> 一致）。

### Long Cang — `LongCang.ttf`

- 实际字体/版本：Long Cang Regular，Version 2.001。
- 二进制版权：`Copyright 2018 The LongCang Project Authors (https://github.com/googlefonts/longcang)`。
- OFL 版权：`Copyright 2018 The Long Cang Project Authors (https://github.com/googlefonts/longcang)`。
- 官方分发及哈希比对来源：<https://raw.githubusercontent.com/google/fonts/main/ofl/longcang/LongCang-Regular.ttf>。
- 本地 SHA-256：`e5bf2c3f24ef2327c6f136d8f73e2f9dfdf44896fdbeb35a9515f44777bb91bc`。
- 许可证文件：[`assets/fonts/licenses/LongCang-OFL.txt`](assets/fonts/licenses/LongCang-OFL.txt)；上游原文：<https://raw.githubusercontent.com/google/fonts/main/ofl/longcang/OFL.txt>。

### Ma Shan Zheng — `MaShanZheng.ttf`

- 实际字体/版本：Ma Shan Zheng Regular，Version 2.003。
- 二进制版权：`Copyright 2018 The MaShanZheng Project Authors (https://github.com/googlefonts/mashanzheng)`。
- OFL 版权：`Copyright 2018 The Ma Shan Zheng Project Authors (https://github.com/googlefonts/mashanzheng)`。
- 官方分发及哈希比对来源：<https://raw.githubusercontent.com/google/fonts/main/ofl/mashanzheng/MaShanZheng-Regular.ttf>。
- 本地 SHA-256：`6d2546bb189c732a8ca29af9e22457b152387d158aa459e4ac2ce1e51788b7fb`。
- 许可证文件：[`assets/fonts/licenses/MaShanZheng-OFL.txt`](assets/fonts/licenses/MaShanZheng-OFL.txt)；上游原文：<https://raw.githubusercontent.com/google/fonts/main/ofl/mashanzheng/OFL.txt>。

### ZCOOL XiaoWei — `ZCOOLXiaoWei.ttf`

- 实际字体/版本：ZCOOL XiaoWei Regular，Version 1.000。
- 二进制版权：`Copyright 2018 The ZCOOL XiaoWei Project Authors (https://www.github.com/googlefonts/xiaowei)`。
- OFL 版权：`Copyright 2018 The ZCOOL XiaoWei Project Authors (https://github.com/googlefonts/zcool-xiaowei/)`。
- 官方分发及哈希比对来源：<https://raw.githubusercontent.com/google/fonts/main/ofl/zcoolxiaowei/ZCOOLXiaoWei-Regular.ttf>。
- 本地 SHA-256：`a42b620140f493db42f741351dfbf343c0936d58588ee8004b8b2a218d997ff1`。
- 许可证文件：[`assets/fonts/licenses/ZCOOLXiaoWei-OFL.txt`](assets/fonts/licenses/ZCOOLXiaoWei-OFL.txt)；上游原文：<https://raw.githubusercontent.com/google/fonts/main/ofl/zcoolxiaowei/OFL.txt>。

### ZCOOL KuaiLe — `ZCOOLKuaiLe.ttf`

- 实际字体/版本：ZCOOL KuaiLe Regular，Version 2.001。
- 二进制版权：`Copyright 2018 The ZCOOL KuaiLe Project Authors (https://github.com/googlefonts/zcool-kuaile)`。
- OFL 版权：`Copyright 2018 The ZCOOL KuaiLe Project Authors (https://github.com/googlefonts/zcool-kuaile)`。
- 官方分发及哈希比对来源：<https://raw.githubusercontent.com/google/fonts/main/ofl/zcoolkuaile/ZCOOLKuaiLe-Regular.ttf>。
- 本地 SHA-256：`812a6fc1fe54b6d73a419245c32dfeba8aa33104d5be90d1cf6af082007cb71d`。
- 许可证文件：[`assets/fonts/licenses/ZCOOLKuaiLe-OFL.txt`](assets/fonts/licenses/ZCOOLKuaiLe-OFL.txt)；上游原文：<https://raw.githubusercontent.com/google/fonts/main/ofl/zcoolkuaile/OFL.txt>。

### JetBrains Mono — `JetBrainsMono.ttf`

- 实际字体/版本：**JetBrains Mono Bold**，Version 2.305（不是 Regular）。
- 二进制及 OFL 版权：`Copyright 2020 The JetBrains Mono Project Authors (https://github.com/JetBrains/JetBrainsMono)`。
- 官方项目及哈希比对来源：<https://raw.githubusercontent.com/JetBrains/JetBrainsMono/master/fonts/ttf/JetBrainsMono-Bold.ttf>。
- 本地 SHA-256：`d22c4f3821d725eb01210d278d95dfcfcaadc34699a06658d47c8a5cc5830ada`。
- 许可证文件：[`assets/fonts/licenses/JetBrainsMono-OFL.txt`](assets/fonts/licenses/JetBrainsMono-OFL.txt)；上游原文：<https://raw.githubusercontent.com/google/fonts/main/ofl/jetbrainsmono/OFL.txt>。

### Montserrat — `Montserrat.ttf`

- 实际字体/版本：**Montserrat Thin**，Version 9.000（不是 Regular 字重）。
- 二进制版权：`Copyright 2011 The Montserrat Project Authors (https://github.com/JulietaUla/Montserrat)`。
- OFL 版权：`Copyright 2024 The Montserrat.Git Project Authors (https://github.com/JulietaUla/Montserrat.git)`。
- 官方项目：<https://github.com/JulietaUla/Montserrat>。
- Version 9.000 官方分发记录：<https://github.com/google/fonts/commit/76fca9fd0bb4ea46583f92e978660f3984ab9442>；该记录标明上游提交为 <https://github.com/JulietaUla/Montserrat/commit/f025b649a5302db60edb04a88199b8b941abc33a>。
- 本地 SHA-256：`b62aede1499c9ea969ebcb8056cc5b386795f7da0912e5b976fe736cd852024f`。
- 许可证文件：[`assets/fonts/licenses/Montserrat-OFL.txt`](assets/fonts/licenses/Montserrat-OFL.txt)；Version 9.000 对应上游原文：<https://raw.githubusercontent.com/google/fonts/76fca9fd0bb4ea46583f92e978660f3984ab9442/ofl/montserrat/OFL.txt>。

### Playfair Display — `PlayfairDisplay.ttf`

- 实际字体/版本：Playfair Display Regular，Version 1.203。
- 二进制版权：`Copyright 2017 The Playfair Display Project Authors (https://github.com/clauseggers/Playfair-Display), with Reserved Font Name "Playfair Display".`
- OFL 版权：`Copyright 2017 The Playfair Display Project Authors (https://github.com/clauseggers/Playfair-Display), with Reserved Font Name "Playfair Display"`。
- 官方项目：<https://github.com/clauseggers/Playfair>（字体元数据中保留了原仓库名 `Playfair-Display`）。
- Version 1.203 官方分发记录：<https://github.com/google/fonts/commit/1e1aa08e994ff7db50116e86ccc7b52a4e4ae5b8>。
- 本地 SHA-256：`c40f2293766a503bc70cce9e512ef844a4ccb7cbcde792fe2ea31d191917d8d6`。
- 许可证文件：[`assets/fonts/licenses/PlayfairDisplay-OFL.txt`](assets/fonts/licenses/PlayfairDisplay-OFL.txt)；Version 1.203 对应上游原文：<https://raw.githubusercontent.com/google/fonts/1e1aa08e994ff7db50116e86ccc7b52a4e4ae5b8/ofl/playfairdisplay/OFL.txt>。
- Reserved Font Name：`Playfair Display`。

### Noto Sans SC — `NotoSansSC-400.ttf`

- 实际字体/版本：Noto Sans SC Regular，Version 2.004-H2。
- 二进制版权：`(c) 2014-2021 Adobe (http://www.adobe.com/), with Reserved Font Name 'Source'.`
- OFL 版权：`Copyright 2014-2021 Adobe (http://www.adobe.com/), with Reserved Font Name 'Source'`。
- 官方分发及哈希比对来源：<https://fonts.gstatic.com/s/notosanssc/v41/k3kCo84MPvpLmixcA63oeAL7Iqp5IZJF9bmaG9_FnYw.ttf>。
- 本地 SHA-256：`450625c8d46ab3df97b7904ded955ec2746d17ec76740cb1e91d1ba63a0f89af`。
- 许可证文件：[`assets/fonts/licenses/NotoSansSC-OFL.txt`](assets/fonts/licenses/NotoSansSC-OFL.txt)；上游原文：<https://raw.githubusercontent.com/google/fonts/main/ofl/notosanssc/OFL.txt>。
- Reserved Font Name：`Source`。

### Noto Sans SC Medium — `NotoSansSC-500.ttf`

- 实际字体/版本：Noto Sans SC Medium，Version 2.004-H2。
- 二进制版权：`(c) 2014-2021 Adobe (http://www.adobe.com/), with Reserved Font Name 'Source'.`
- OFL 版权：`Copyright 2014-2021 Adobe (http://www.adobe.com/), with Reserved Font Name 'Source'`。
- 官方分发及哈希比对来源：<https://fonts.gstatic.com/s/notosanssc/v41/k3kCo84MPvpLmixcA63oeAL7Iqp5IZJF9bmaG-3FnYw.ttf>。
- 本地 SHA-256：`09f17deea660cdd9cc2ba1e8b90adad18eabeca73b62b3c5f76b7ec3b61b33e0`。
- 许可证文件：与 Noto Sans SC 同族共用 [`assets/fonts/licenses/NotoSansSC-OFL.txt`](assets/fonts/licenses/NotoSansSC-OFL.txt)。
- Reserved Font Name：`Source`。

### Noto Sans SC Bold — `NotoSansSC-700.ttf`

- 实际字体/版本：Noto Sans SC Bold，Version 2.004-H2。
- 二进制版权：`(c) 2014-2021 Adobe (http://www.adobe.com/), with Reserved Font Name 'Source'.`
- OFL 版权：`Copyright 2014-2021 Adobe (http://www.adobe.com/), with Reserved Font Name 'Source'`。
- 官方分发及哈希比对来源：<https://fonts.gstatic.com/s/notosanssc/v41/k3kCo84MPvpLmixcA63oeAL7Iqp5IZJF9bmaGzjCnYw.ttf>。
- 本地 SHA-256：`0066a522a1ac007c1d72bc4fccb114f80ff7294641c78cead9715bd14d43b9ea`。
- 许可证文件：与 Noto Sans SC 同族共用 [`assets/fonts/licenses/NotoSansSC-OFL.txt`](assets/fonts/licenses/NotoSansSC-OFL.txt)。
- Reserved Font Name：`Source`。

## 2. Go modules

版本来自 `go.mod`；许可证全文核对自本机 Go module cache 中对应版本的原始文件。下表中的链接指向上游项目或对应许可证。

| 模块 | 版本 | 许可证及通知 |
| --- | --- | --- |
| `github.com/6tail/lunar-go` | `v1.4.7` | MIT；Copyright (c) 2020 6tail。全文见下。<https://github.com/6tail/lunar-go/blob/v1.4.7/LICENSE> |
| `golang.org/x/image` | `v0.48.0` | BSD-3-Clause；Copyright 2009 The Go Authors；另有 PATENTS 授权。全文见下。<https://github.com/golang/image/blob/v0.48.0/LICENSE> |
| `golang.org/x/sys` | `v0.50.0` | BSD-3-Clause；Copyright 2009 The Go Authors；另有 PATENTS 授权。全文见下。<https://github.com/golang/sys/blob/v0.50.0/LICENSE> |
| `golang.org/x/text` | `v0.44.0` | BSD-3-Clause；Copyright 2009 The Go Authors；另有 PATENTS 授权。全文见下。<https://github.com/golang/text/blob/v0.44.0/LICENSE> |

三个 `golang.org/x` 模块对应版本的 `LICENSE` 文件彼此完全相同，`PATENTS` 文件也彼此完全相同，因此在下文中各保留一份共同文本，适用于这三个模块。

### `github.com/6tail/lunar-go` — MIT License

```text
MIT License

Copyright (c) 2020 6tail

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

### `golang.org/x/image`, `golang.org/x/sys`, `golang.org/x/text` — BSD-3-Clause

```text
Copyright 2009 The Go Authors.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google LLC nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

### `golang.org/x/image`, `golang.org/x/sys`, `golang.org/x/text` — PATENTS

```text
Additional IP Rights Grant (Patents)

"This implementation" means the copyrightable works distributed by
Google as part of the Go project.

Google hereby grants to You a perpetual, worldwide, non-exclusive,
no-charge, royalty-free, irrevocable (except as stated in this section)
patent license to make, have made, use, offer to sell, sell, import,
transfer and otherwise run, modify and propagate the contents of this
implementation of Go, where such license applies only to those patent
claims, both currently owned or controlled by Google and acquired in
the future, licensable by Google that are necessarily infringed by this
implementation of Go.  This grant does not include claims that would be
infringed only as a consequence of further modification of this
implementation.  If you or your agent or exclusive licensee institute or
order or agree to the institution of patent litigation against any
entity (including a cross-claim or counterclaim in a lawsuit) alleging
that this implementation of Go or any code incorporated within this
implementation of Go constitutes direct or contributory patent
infringement, or inducement of patent infringement, then any patent
rights granted to you under this License for this implementation of Go
shall terminate as of the date such litigation is filed.
```
