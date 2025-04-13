export declare const GlobalPrefixError = "missing prefix '{{Infobox' at the start";
export declare const GlobalSuffixError = "missing suffix '}}' at the end";
export declare const ArrayNoCloseError = "array should be closed by '}'";
export declare const ArrayItemWrappedError = "array item should be wrapped by '[]'";
export declare const ExpectingNewFieldError = "missing '|' to start a new field";
export declare const ExpectingSignEqualError = "missing '=' to separate field name and value";
export declare class WikiSyntaxError extends Error {
    lino: number;
    line: string | null;
    constructor(lino: number, line: string | null, message: string);
}
//# sourceMappingURL=error.d.ts.map