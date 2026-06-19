# pathfinder-ui

Pathfinder 前端应用，基于 Next.js 构建。

## Tech Stack

| 类别        | 技术                                                                    |
| ----------- | ----------------------------------------------------------------------- |
| 框架        | [Next.js](https://nextjs.org) 15 (App Router, Turbopack)                |
| UI 库       | [React](https://react.dev) 19                                           |
| 语言        | TypeScript 5                                                            |
| 样式        | [Tailwind CSS](https://tailwindcss.com) v4                              |
| UI 组件     | [shadcn/ui](https://ui.shadcn.com)（基于 Radix UI 原语）                |
| 图标        | [Lucide React](https://lucide.dev)                                      |
| 服务端状态  | [TanStack Query (React Query)](https://tanstack.com/query) v5           |
| HTTP 客户端 | [axios](https://axios-http.com)                                         |
| 表单        | [react-hook-form](https://react-hook-form.com) + [Zod](https://zod.dev) |
| 拖拽排序    | [dnd-kit](https://dndkit.com)                                           |
| 通知        | [sonner](https://sonner.emilkowal.ski)                                  |
| E2E 测试    | [Playwright](https://playwright.dev)                                    |
| 包管理器    | [pnpm](https://pnpm.io)                                                 |

## Getting Started

```bash
pnpm install
pnpm dev
```

Open [http://localhost:3000](http://localhost:3000) with your browser to see the result.

## Scripts

```bash
pnpm dev          # 启动开发服务器（Turbopack）
pnpm build        # 生产构建
pnpm start        # 启动生产服务器
pnpm lint         # 代码检查
pnpm test:e2e     # 运行 Playwright E2E 测试
pnpm test:e2e:ui  # 以交互模式运行 E2E 测试
```
