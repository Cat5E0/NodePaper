# 公式与图片压力 {#sec:layout-formula}

NP-LAYOUT-EQUATION-01。长公式采用可换行的对齐环境：

$$
\begin{aligned}
F(\boldsymbol{x})={}&\sum_{i=1}^{n}\sum_{j=1}^{m} c_{ij}x_{ij}
+\lambda_1\sum_{i=1}^{n}(x_i-\bar{x})^2 \\
&+\lambda_2\sum_{t=1}^{T}\left\lVert
\boldsymbol{A}_t\boldsymbol{x}_t-\boldsymbol{b}_t
\right\rVert_2^2.
\end{aligned}
$$ {#eq:layout-aligned}

矩阵与分段函数如下：

$$
\mathbf{M}=\begin{bmatrix}
1&2&3&4\\
5&6&7&8\\
9&10&11&12\\
13&14&15&16
\end{bmatrix},\qquad
f(x)=\begin{cases}
x^2,&x\ge 0,\\
-x+1,&x<0.
\end{cases}
$$

受控公式 Fragment：

\input{equations/long-objective.tex}

式 \eqref{eq:fragment-objective} 是 Fragment 中的稳定标签。

受控 TikZ Fragment：

\input{figures/tikz-diagram.tex}

下图是绘图工具直接产出的 PGF 代码，以 `.pgf` 声明为 Fragment：

\input{figures/mpl-plot.pgf}

图 \ref{fig:layout-tikz} 由 Fragment 内的 tikzpicture 绘制，其中的 TikZ 库由 Fragment 自行载入。

![NP-LAYOUT-IMAGE-01 极端宽高比图片及较长图题，用于确认图片按比例限制在正文区域内且图题可以自然换行](images/extreme-wide.png){#fig:layout-wide width=100%}

图 @fig:layout-wide 与式 @eq:layout-aligned 均应可以解析。
