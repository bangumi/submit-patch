export interface Wiki {
    type: string;
    data: WikiItem[];
}
/** JS 的 map 会按照插入顺序排序 */
export interface WikiMap {
    type: string;
    data: Map<string, WikiItem>;
}
export type WikiItemType = 'array' | 'object';
export declare class WikiArrayItem {
    k?: string;
    v?: string;
    constructor(k?: string, v?: string);
}
export declare class WikiItem {
    key: string;
    value?: string;
    array?: boolean;
    values?: WikiArrayItem[];
    constructor(key: string, value: string, type: WikiItemType);
    private convertToArray;
    push(item: WikiItem): void;
}
//# sourceMappingURL=types.d.ts.map