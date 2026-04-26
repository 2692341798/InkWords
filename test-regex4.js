const md = `**注意**: Mermaid 图只用于展示趋势，实际曲线可参考下图（摘自 hello-algo）:

![复杂度增长曲线](https://www.hello-
algo.com/chapter_computational_complexity/time_complexity.asse
ts/time_complexity_curve.png)

Other text.
`;

const fixedMd = md.replace(/\]\(([^)]+)\)/g, (match, p1) => {
  return `](${p1.replace(/\s*\n\s*/g, '')})`;
});

console.log(fixedMd);
