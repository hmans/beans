import { Node, mergeAttributes } from '@tiptap/core';

/**
 * Inline atom node for file mentions in the agent chat composer.
 * Renders as a styled pill showing the file path. Non-editable —
 * backspace deletes the entire node as a unit.
 */
export const FileMention = Node.create({
  name: 'fileMention',
  group: 'inline',
  inline: true,
  atom: true,

  addAttributes() {
    return {
      path: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('data-path'),
        renderHTML: (attributes: Record<string, string>) => ({ 'data-path': attributes.path })
      }
    };
  },

  parseHTML() {
    return [{ tag: 'span[data-file-mention]' }];
  },

  renderHTML({ node, HTMLAttributes }) {
    return [
      'span',
      mergeAttributes(HTMLAttributes, {
        'data-file-mention': '',
        class: 'file-mention-pill',
        contenteditable: 'false'
      }),
      node.attrs.path
    ];
  }
});
