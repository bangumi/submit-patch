import type { Wiki, WikiMap } from './types.js';
export * from './types.js';
export * from './error.js';
export { stringify, stringifyMap } from './stringify.js';
/** 解析 wiki 文本，以 `Map` 类型返回解析结果。 会合并重复出现的 key */
export declare function parseToMap(s: string): WikiMap;
export declare function parse(s: string): Wiki;
//# sourceMappingURL=index.d.ts.map