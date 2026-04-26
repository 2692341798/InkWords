import React from 'react';
import { renderToString } from 'react-dom/server';
import ReactMarkdown from 'react-markdown';

const md = `![复杂度增长曲线](https://www.hello-
algo.com/chapter_computational_complexity/time_complexity.asse
ts/time_complexity_curve.png)`;

const html = renderToString(
  React.createElement(ReactMarkdown, {
    children: md,
    components: {
      img(props) {
        console.log("Parsed img src:", JSON.stringify(props.src));
        return React.createElement('img', props as any);
      }
    }
  })
);
console.log(html);
