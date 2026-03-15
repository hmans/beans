import type { CodegenConfig } from '@graphql-codegen/cli';

// Inline type definition prepended to the generated file.
// The codegen plugin normally imports from @graphql-typed-document-node/core,
// but that package has "main": "" which breaks Vite's module resolution.
// Instead we import from urql (which re-exports its own compatible type) and
// strip the codegen's broken import via a sed hook.
const documentNodeTypePreamble = `\
import type { TypedDocumentNode as DocumentNode } from 'urql';
`;

const config: CodegenConfig = {
  schema: '../internal/graph/schema.graphqls',
  documents: 'src/lib/graphql/operations.graphql',
  generates: {
    'src/lib/graphql/generated.ts': {
      plugins: [
        { add: { content: documentNodeTypePreamble } },
        'typescript',
        'typescript-operations',
        'typed-document-node',
      ],
      config: {
        // Use 'string' for the Time scalar (ISO 8601 strings from the backend)
        scalars: {
          Time: 'string',
        },
        // Avoid __typename pollution since urql doesn't require it
        skipTypename: true,
        // Use 'type' instead of 'interface' for better compatibility
        declarationKind: 'type',
        // Prevent the typed-document-node plugin from importing from
        // @graphql-typed-document-node/core. We prepend our own definition above.
        documentNodeImport: '__PLACEHOLDER_DO_NOT_IMPORT__#TypedDocumentNode',
      },
    },
  },
  hooks: {
    afterOneFileWrite: [
      // Remove the placeholder import that the plugin generates
      'perl -i -ne \'print unless /^import.*__PLACEHOLDER_DO_NOT_IMPORT__/\'',
    ],
  },
};

export default config;
