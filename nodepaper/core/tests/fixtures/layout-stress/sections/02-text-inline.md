# 长文本与行内对象 {#sec:layout-text}

NP-LAYOUT-TEXT-01。这是一段包含较长中文说明的虚构文本，用来确认普通段落能够在页面边界内自然换行，并且不会因为连续出现的中文、English words、数字 2026202620262026 和标点符号而静默越过正文区域。The corresponding English paragraph contains enough ordinary words to exercise line breaking without relying on a visual pixel golden, while preserving the stable marker NP-LAYOUT-TEXT-EN-01 for extraction.

长 URL 使用可断行链接：<https://example.invalid/nodepaper/layout/stress/a-very-long-but-entirely-fictional-resource-path?profile=cumcm&year=2026&mode=electronic-paper>。

DOI 示例为 <https://doi.org/10.0000/nodepaper.layout.stress.2026>。Windows 路径示例为 `C:\Users\Student\Documents\NodePaper\layout-stress\sections\02-text-inline.md`，行内代码示例为 `nodepaper build D:\papers\cumcm-layout-stress`。

行内代码内的中文必须保留字形，反斜杠不得被改写：中文路径示例为 `D:\论文样例\第一章\模型求解.md`，中英混排命令示例为 `nodepaper build D:\论文\2026建模`。
