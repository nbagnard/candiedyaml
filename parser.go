package candiedyaml

import (
	"bytes"
)

/*
 * The parser implements the following grammar:
 *
 * stream               ::= STREAM-START implicit_document? explicit_document* STREAM-END
 * implicit_document    ::= block_node DOCUMENT-END*
 * explicit_document    ::= DIRECTIVE* DOCUMENT-START block_node? DOCUMENT-END*
 * block_node_or_indentless_sequence    ::=
 *                          ALIAS
 *                          | properties (block_content | indentless_block_sequence)?
 *                          | block_content
 *                          | indentless_block_sequence
 * block_node           ::= ALIAS
 *                          | properties block_content?
 *                          | block_content
 * flow_node            ::= ALIAS
 *                          | properties flow_content?
 *                          | flow_content
 * properties           ::= TAG ANCHOR? | ANCHOR TAG?
 * block_content        ::= block_collection | flow_collection | SCALAR
 * flow_content         ::= flow_collection | SCALAR
 * block_collection     ::= block_sequence | block_mapping
 * flow_collection      ::= flow_sequence | flow_mapping
 * block_sequence       ::= BLOCK-SEQUENCE-START (BLOCK-ENTRY block_node?)* BLOCK-END
 * indentless_sequence  ::= (BLOCK-ENTRY block_node?)+
 * block_mapping        ::= BLOCK-MAPPING_START
 *                          ((KEY block_node_or_indentless_sequence?)?
 *                          (VALUE block_node_or_indentless_sequence?)?)*
 *                          BLOCK-END
 * flow_sequence        ::= FLOW-SEQUENCE-START
 *                          (flow_sequence_entry FLOW-ENTRY)*
 *                          flow_sequence_entry?
 *                          FLOW-SEQUENCE-END
 * flow_sequence_entry  ::= flow_node | KEY flow_node? (VALUE flow_node?)?
 * flow_mapping         ::= FLOW-MAPPING-START
 *                          (flow_mapping_entry FLOW-ENTRY)*
 *                          flow_mapping_entry?
 *                          FLOW-MAPPING-END
 * flow_mapping_entry   ::= flow_node | KEY flow_node? (VALUE flow_node?)?
 */

/*
 * Peek the next token in the token queue.
 */
func peek_token(parser *yaml_parser_t) *yaml_token_t {
	if parser.token_available || yaml_parser_fetch_more_tokens(parser) {
		return &parser.tokens[parser.tokens_head]
	}
	return nil
}

/*
 * Remove the next token from the queue (must be called after peek_token).
 */
func skip_token(parser *yaml_parser_t) {
	parser.token_available = false
	parser.tokens_parsed++
	parser.stream_end_produced = parser.tokens[parser.tokens_head].token_type == YAML_STREAM_END_TOKEN
	parser.tokens_head++
}

/*
 * Get the next event.
 */

func yaml_parser_parse(parser *yaml_parser_t, event *yaml_event_t) bool {
	/* Erase the event object. */
	*event = yaml_event_t{}

	/* No events after the end of the stream or error. */

	if parser.stream_end_produced || parser.error != YAML_NO_ERROR ||
		parser.state == YAML_PARSE_END_STATE {
		return true
	}

	/* Generate the next event. */

	return yaml_parser_state_machine(parser, event)
}

/*
 * Set parser error.
 */

func yaml_parser_set_parser_error(parser *yaml_parser_t,
	problem string, problem_mark yaml_mark_t) bool {
	parser.error = YAML_PARSER_ERROR
	parser.problem = problem
	parser.problem_mark = problem_mark

	return false
}

func yaml_parser_set_parser_error_context(parser *yaml_parser_t,
	context string, context_mark yaml_mark_t,
	problem string, problem_mark yaml_mark_t) bool {
	parser.error = YAML_PARSER_ERROR
	parser.context = context
	parser.context_mark = context_mark
	parser.problem = problem
	parser.problem_mark = problem_mark

	return false
}

/*
 * State dispatcher.
 */

func yaml_parser_state_machine(parser *yaml_parser_t, event *yaml_event_t) bool {
	switch parser.state {
	case YAML_PARSE_STREAM_START_STATE:
		return yaml_parser_parse_stream_start(parser, event)

	case YAML_PARSE_IMPLICIT_DOCUMENT_START_STATE:
		return yaml_parser_parse_document_start(parser, event, true)

	case YAML_PARSE_DOCUMENT_START_STATE:
		return yaml_parser_parse_document_start(parser, event, false)

	case YAML_PARSE_DOCUMENT_CONTENT_STATE:
		return yaml_parser_parse_document_content(parser, event)

	case YAML_PARSE_DOCUMENT_END_STATE:
		return yaml_parser_parse_document_end(parser, event)

	case YAML_PARSE_BLOCK_NODE_STATE:
		return yaml_parser_parse_node(parser, event, true, false)

	case YAML_PARSE_BLOCK_NODE_OR_INDENTLESS_SEQUENCE_STATE:
		return yaml_parser_parse_node(parser, event, true, true)

	case YAML_PARSE_FLOW_NODE_STATE:
		return yaml_parser_parse_node(parser, event, false, false)

	case YAML_PARSE_BLOCK_SEQUENCE_FIRST_ENTRY_STATE:
		return yaml_parser_parse_block_sequence_entry(parser, event, true)

	case YAML_PARSE_BLOCK_SEQUENCE_ENTRY_STATE:
		return yaml_parser_parse_block_sequence_entry(parser, event, false)

	case YAML_PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE:
		return yaml_parser_parse_indentless_sequence_entry(parser, event)

	case YAML_PARSE_BLOCK_MAPPING_FIRST_KEY_STATE:
		return yaml_parser_parse_block_mapping_key(parser, event, true)

	case YAML_PARSE_BLOCK_MAPPING_KEY_STATE:
		return yaml_parser_parse_block_mapping_key(parser, event, false)

	case YAML_PARSE_BLOCK_MAPPING_VALUE_STATE:
		return yaml_parser_parse_block_mapping_value(parser, event)

	case YAML_PARSE_FLOW_SEQUENCE_FIRST_ENTRY_STATE:
		return yaml_parser_parse_flow_sequence_entry(parser, event, true)

	case YAML_PARSE_FLOW_SEQUENCE_ENTRY_STATE:
		return yaml_parser_parse_flow_sequence_entry(parser, event, false)

	case YAML_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_KEY_STATE:
		return yaml_parser_parse_flow_sequence_entry_mapping_key(parser, event)

	case YAML_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_VALUE_STATE:
		return yaml_parser_parse_flow_sequence_entry_mapping_value(parser, event)

	case YAML_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_END_STATE:
		return yaml_parser_parse_flow_sequence_entry_mapping_end(parser, event)

	case YAML_PARSE_FLOW_MAPPING_FIRST_KEY_STATE:
		return yaml_parser_parse_flow_mapping_key(parser, event, true)

	case YAML_PARSE_FLOW_MAPPING_KEY_STATE:
		return yaml_parser_parse_flow_mapping_key(parser, event, false)

	case YAML_PARSE_FLOW_MAPPING_VALUE_STATE:
		return yaml_parser_parse_flow_mapping_value(parser, event, false)

	case YAML_PARSE_FLOW_MAPPING_EMPTY_VALUE_STATE:
		return yaml_parser_parse_flow_mapping_value(parser, event, true)
	}

	panic("invalid parser state")
}

/*
 * Parse the production:
 * stream   ::= STREAM-START implicit_document? explicit_document* STREAM-END
 *              ************
 */

func yaml_parser_parse_stream_start(parser *yaml_parser_t, event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}

	if token.token_type != YAML_STREAM_START_TOKEN {
		return yaml_parser_set_parser_error(parser,
			"did not find expected <stream-start>", token.start_mark)
	}

	parser.state = YAML_PARSE_IMPLICIT_DOCUMENT_START_STATE
	*event = yaml_event_t{
		event_type: YAML_STREAM_START_EVENT,
		start_mark: token.start_mark,
		end_mark:   token.end_mark,
		encoding:   token.encoding,
	}
	skip_token(parser)

	return true
}

/*
 * Parse the productions:
 * implicit_document    ::= block_node DOCUMENT-END*
 *                          *
 * explicit_document    ::= DIRECTIVE* DOCUMENT-START block_node? DOCUMENT-END*
 *                          *************************
 */

func yaml_parser_parse_document_start(parser *yaml_parser_t, event *yaml_event_t,
	implicit bool) bool {

	token := peek_token(parser)
	if token == nil {
		return false
	}

	/* Parse extra document end indicators. */

	if !implicit {
		for token.token_type == YAML_DOCUMENT_END_TOKEN {
			skip_token(parser)
			token = peek_token(parser)
			if token == nil {
				return false
			}
		}
	}

	/* Parse an implicit document. */

	if implicit && token.token_type != YAML_VERSION_DIRECTIVE_TOKEN &&
		token.token_type != YAML_TAG_DIRECTIVE_TOKEN &&
		token.token_type != YAML_DOCUMENT_START_TOKEN &&
		token.token_type != YAML_STREAM_END_TOKEN {
		if !yaml_parser_process_directives(parser, nil, nil) {
			return false
		}

		parser.states = append(parser.states, YAML_PARSE_DOCUMENT_END_STATE)
		parser.state = YAML_PARSE_BLOCK_NODE_STATE

		*event = yaml_event_t{
			event_type: YAML_DOCUMENT_START_EVENT,
			implicit:   true,
			start_mark: token.start_mark,
			end_mark:   token.end_mark,
		}
	} else if token.token_type != YAML_STREAM_END_TOKEN {
		/* Parse an explicit document. */
		var version_directive *yaml_version_directive_t
		var tag_directives []yaml_tag_directive_t

		start_mark := token.start_mark
		if !yaml_parser_process_directives(parser, &version_directive,
			&tag_directives) {
			return false
		}
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.token_type != YAML_DOCUMENT_START_TOKEN {
			yaml_parser_set_parser_error(parser,
				"did not find expected <document start>", token.start_mark)
			return false
		}

		parser.states = append(parser.states, YAML_PARSE_DOCUMENT_END_STATE)
		parser.state = YAML_PARSE_DOCUMENT_CONTENT_STATE

		end_mark := token.end_mark

		*event = yaml_event_t{
			event_type:        YAML_DOCUMENT_START_EVENT,
			start_mark:        start_mark,
			end_mark:          end_mark,
			version_directive: version_directive,
			tag_directives:    tag_directives,
			implicit:          false,
		}
		skip_token(parser)
	} else {
		/* Parse the stream end. */
		parser.state = YAML_PARSE_END_STATE

		*event = yaml_event_t{
			event_type: YAML_STREAM_END_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.end_mark,
		}
		skip_token(parser)
	}
	return true
}

/*
 * Parse the productions:
 * explicit_document    ::= DIRECTIVE* DOCUMENT-START block_node? DOCUMENT-END*
 *                                                    ***********
 */

func yaml_parser_parse_document_content(parser *yaml_parser_t, event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}

	if token.token_type == YAML_VERSION_DIRECTIVE_TOKEN ||
		token.token_type == YAML_TAG_DIRECTIVE_TOKEN ||
		token.token_type == YAML_DOCUMENT_START_TOKEN ||
		token.token_type == YAML_DOCUMENT_END_TOKEN ||
		token.token_type == YAML_STREAM_END_TOKEN {
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]
		return yaml_parser_process_empty_scalar(parser, event,
			token.start_mark)
	} else {
		return yaml_parser_parse_node(parser, event, true, false)
	}
}

/*
 * Parse the productions:
 * implicit_document    ::= block_node DOCUMENT-END*
 *                                     *************
 * explicit_document    ::= DIRECTIVE* DOCUMENT-START block_node? DOCUMENT-END*
 *                                                                *************
 */

func yaml_parser_parse_document_end(parser *yaml_parser_t, event *yaml_event_t) bool {
	implicit := true

	token := peek_token(parser)
	if token == nil {
		return false
	}

	start_mark, end_mark := token.start_mark, token.start_mark

	if token.token_type == YAML_DOCUMENT_END_TOKEN {
		end_mark = token.end_mark
		skip_token(parser)
		implicit = false
	}

	parser.tag_directives = parser.tag_directives[:0]

	parser.state = YAML_PARSE_DOCUMENT_START_STATE
	*event = yaml_event_t{
		event_type: YAML_DOCUMENT_END_EVENT,
		start_mark: start_mark,
		end_mark:   end_mark,
		implicit:   implicit,
	}

	return true
}

/*
 * Parse the productions:
 * block_node_or_indentless_sequence    ::=
 *                          ALIAS
 *                          *****
 *                          | properties (block_content | indentless_block_sequence)?
 *                            **********  *
 *                          | block_content | indentless_block_sequence
 *                            *
 * block_node           ::= ALIAS
 *                          *****
 *                          | properties block_content?
 *                            ********** *
 *                          | block_content
 *                            *
 * flow_node            ::= ALIAS
 *                          *****
 *                          | properties flow_content?
 *                            ********** *
 *                          | flow_content
 *                            *
 * properties           ::= TAG ANCHOR? | ANCHOR TAG?
 *                          *************************
 * block_content        ::= block_collection | flow_collection | SCALAR
 *                                                               ******
 * flow_content         ::= flow_collection | SCALAR
 *                                            ******
 */

func yaml_parser_parse_node(parser *yaml_parser_t, event *yaml_event_t,
	block bool, indentless_sequence bool) bool {

	token := peek_token(parser)
	if token == nil {
		return false
	}

	if token.token_type == YAML_ALIAS_TOKEN {
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]

		*event = yaml_event_t{
			event_type: YAML_ALIAS_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.end_mark,
			anchor:     token.value,
		}
		skip_token(parser)
		return true
	} else {
		start_mark, end_mark := token.start_mark, token.start_mark

		var tag_handle *[]byte
		var tag_suffix, anchor []byte
		var tag_mark yaml_mark_t
		if token.token_type == YAML_ANCHOR_TOKEN {
			anchor = token.value
			start_mark = token.start_mark
			end_mark = token.end_mark
			skip_token(parser)
			token = peek_token(parser)
			if token == nil {
				return false
			}
			if token.token_type == YAML_TAG_TOKEN {
				*tag_handle = token.value
				tag_suffix = token.suffix
				tag_mark = token.start_mark
				end_mark = token.end_mark
				skip_token(parser)
				token = peek_token(parser)
				if token == nil {
					return false
				}
			}
		} else if token.token_type == YAML_TAG_TOKEN {
			*tag_handle = token.value
			tag_suffix = token.suffix
			start_mark, tag_mark = token.start_mark, token.start_mark
			end_mark = token.end_mark
			skip_token(parser)
			token = peek_token(parser)
			if token == nil {
				return false
			}
			if token.token_type == YAML_ANCHOR_TOKEN {
				anchor = token.value
				end_mark = token.end_mark
				skip_token(parser)
				token = peek_token(parser)
				if token == nil {
					return false
				}

			}
		}

		var tag []byte
		if tag_handle != nil {
			if len(*tag_handle) == 0 {
				tag = tag_suffix
				tag_handle = nil
				tag_suffix = nil
			} else {
				for i := range parser.tag_directives {
					tag_directive := &parser.tag_directives[i]
					if bytes.Equal(tag_directive.handle, *tag_handle) {
						tag = append([]byte(nil), tag_directive.prefix...)
						tag = append(tag, tag_suffix...)
						tag_handle = nil
						tag_suffix = nil
						break
					}
				}
				if len(tag) == 0 {
					yaml_parser_set_parser_error_context(parser,
						"while parsing a node", start_mark,
						"found undefined tag handle", tag_mark)
					return false
				}
			}
		}

		implicit := len(tag) == 0
		if indentless_sequence && token.token_type == YAML_BLOCK_ENTRY_TOKEN {
			end_mark = token.end_mark
			parser.state = YAML_PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE

			*event = yaml_event_t{
				event_type: YAML_SEQUENCE_START_EVENT,
				start_mark: start_mark,
				end_mark:   end_mark,
				anchor:     anchor,
				tag:        tag,
				implicit:   implicit,
				style:      yaml_style_t(YAML_BLOCK_SEQUENCE_STYLE),
			}

			return true
		} else {
			if token.token_type == YAML_SCALAR_TOKEN {
				plain_implicit := false
				quoted_implicit := false
				end_mark = token.end_mark
				if (token.style == YAML_PLAIN_SCALAR_STYLE && len(tag) == 0) ||
					(len(tag) == 1 && tag[0] == '!') {
					plain_implicit = true
				} else if len(tag) == 0 {
					quoted_implicit = true
				}

				parser.state = parser.states[len(parser.states)-1]
				parser.states = parser.states[:len(parser.states)-1]

				*event = yaml_event_t{
					event_type:      YAML_SCALAR_EVENT,
					start_mark:      start_mark,
					end_mark:        end_mark,
					anchor:          anchor,
					tag:             tag,
					value:           token.value,
					implicit:        plain_implicit,
					quoted_implicit: quoted_implicit,
					style:           yaml_style_t(token.style),
				}

				skip_token(parser)
				return true
			} else if token.token_type == YAML_FLOW_SEQUENCE_START_TOKEN {
				end_mark = token.end_mark
				parser.state = YAML_PARSE_FLOW_SEQUENCE_FIRST_ENTRY_STATE

				*event = yaml_event_t{
					event_type: YAML_SEQUENCE_START_EVENT,
					start_mark: start_mark,
					end_mark:   end_mark,
					anchor:     anchor,
					tag:        tag,
					implicit:   implicit,
					style:      yaml_style_t(YAML_FLOW_SEQUENCE_STYLE),
				}

				return true
			} else if token.token_type == YAML_FLOW_MAPPING_START_TOKEN {
				end_mark = token.end_mark
				parser.state = YAML_PARSE_FLOW_MAPPING_FIRST_KEY_STATE

				*event = yaml_event_t{
					event_type: YAML_MAPPING_START_EVENT,
					start_mark: start_mark,
					end_mark:   end_mark,
					anchor:     anchor,
					tag:        tag,
					implicit:   implicit,
					style:      yaml_style_t(YAML_FLOW_MAPPING_STYLE),
				}

				return true
			} else if block && token.token_type == YAML_BLOCK_SEQUENCE_START_TOKEN {
				end_mark = token.end_mark
				parser.state = YAML_PARSE_BLOCK_SEQUENCE_FIRST_ENTRY_STATE

				*event = yaml_event_t{
					event_type: YAML_SEQUENCE_START_EVENT,
					start_mark: start_mark,
					end_mark:   end_mark,
					anchor:     anchor,
					tag:        tag,
					implicit:   implicit,
					style:      yaml_style_t(YAML_BLOCK_SEQUENCE_STYLE),
				}

				return true
			} else if block && token.token_type == YAML_BLOCK_MAPPING_START_TOKEN {
				end_mark = token.end_mark
				parser.state = YAML_PARSE_BLOCK_MAPPING_FIRST_KEY_STATE

				*event = yaml_event_t{
					event_type: YAML_MAPPING_START_EVENT,
					start_mark: start_mark,
					end_mark:   end_mark,
					anchor:     anchor,
					tag:        tag,
					implicit:   implicit,
					style:      yaml_style_t(YAML_BLOCK_MAPPING_STYLE),
				}
				return true
			} else if len(anchor) > 0 || len(tag) > 0 {

				parser.state = parser.states[len(parser.states)-1]
				parser.states = parser.states[:len(parser.states)-1]

				*event = yaml_event_t{
					event_type:      YAML_SCALAR_EVENT,
					start_mark:      start_mark,
					end_mark:        end_mark,
					anchor:          anchor,
					tag:             tag,
					implicit:        implicit,
					quoted_implicit: false,
					style:           yaml_style_t(YAML_PLAIN_SCALAR_STYLE),
				}
				return true
			} else {
				msg := "while parsing a block node"
				if !block {
					msg = "while parsing a flow node"
				}
				yaml_parser_set_parser_error_context(parser, msg, start_mark,
					"did not find expected node content", token.start_mark)
				return false
			}
		}
	}

	return false
}

/*
 * Parse the productions:
 * block_sequence ::= BLOCK-SEQUENCE-START (BLOCK-ENTRY block_node?)* BLOCK-END
 *                    ********************  *********** *             *********
 */

func yaml_parser_parse_block_sequence_entry(parser *yaml_parser_t,
	event *yaml_event_t, first bool) bool {
	if first {
		token := peek_token(parser)
		parser.marks = append(parser.marks, token.start_mark)
		skip_token(parser)
	}

	token := peek_token(parser)
	if token == nil {
		return false
	}

	if token.token_type == YAML_BLOCK_ENTRY_TOKEN {
		mark := token.end_mark
		skip_token(parser)
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.token_type != YAML_BLOCK_ENTRY_TOKEN &&
			token.token_type != YAML_BLOCK_END_TOKEN {
			parser.states = append(parser.states, YAML_PARSE_BLOCK_SEQUENCE_ENTRY_STATE)
			return yaml_parser_parse_node(parser, event, true, false)
		} else {
			parser.state = YAML_PARSE_BLOCK_SEQUENCE_ENTRY_STATE
			return yaml_parser_process_empty_scalar(parser, event, mark)
		}
	} else if token.token_type == YAML_BLOCK_END_TOKEN {
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]
		parser.marks = parser.marks[:len(parser.marks)-1]

		*event = yaml_event_t{
			event_type: YAML_SEQUENCE_END_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.end_mark,
		}

		skip_token(parser)
		return true
	} else {
		mark := parser.marks[len(parser.marks)-1]
		parser.marks = parser.marks[:len(parser.marks)-1]

		return yaml_parser_set_parser_error_context(parser,
			"while parsing a block collection", mark,
			"did not find expected '-' indicator", token.start_mark)
	}
}

/*
 * Parse the productions:
 * indentless_sequence  ::= (BLOCK-ENTRY block_node?)+
 *                           *********** *
 */

func yaml_parser_parse_indentless_sequence_entry(parser *yaml_parser_t,
	event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}

	if token.token_type == YAML_BLOCK_ENTRY_TOKEN {
		mark := token.end_mark
		skip_token(parser)
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.token_type != YAML_BLOCK_ENTRY_TOKEN &&
			token.token_type != YAML_KEY_TOKEN &&
			token.token_type != YAML_VALUE_TOKEN &&
			token.token_type != YAML_BLOCK_END_TOKEN {
			parser.states = append(parser.states, YAML_PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE)
			return yaml_parser_parse_node(parser, event, true, false)
		} else {
			parser.state = YAML_PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE
			return yaml_parser_process_empty_scalar(parser, event, mark)
		}
	} else {
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]

		*event = yaml_event_t{
			event_type: YAML_SEQUENCE_END_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.start_mark,
		}
		return true
	}
}

/*
 * Parse the productions:
 * block_mapping        ::= BLOCK-MAPPING_START
 *                          *******************
 *                          ((KEY block_node_or_indentless_sequence?)?
 *                            *** *
 *                          (VALUE block_node_or_indentless_sequence?)?)*
 *
 *                          BLOCK-END
 *                          *********
 */

func yaml_parser_parse_block_mapping_key(parser *yaml_parser_t,
	event *yaml_event_t, first bool) bool {
	if first {
		token := peek_token(parser)
		parser.marks = append(parser.marks, token.start_mark)
		skip_token(parser)
	}

	token := peek_token(parser)
	if token == nil {
		return false
	}

	if token.token_type == YAML_KEY_TOKEN {
		mark := token.end_mark
		skip_token(parser)
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.token_type != YAML_KEY_TOKEN &&
			token.token_type != YAML_VALUE_TOKEN &&
			token.token_type != YAML_BLOCK_END_TOKEN {
			parser.states = append(parser.states, YAML_PARSE_BLOCK_MAPPING_VALUE_STATE)
			return yaml_parser_parse_node(parser, event, true, true)
		} else {
			parser.state = YAML_PARSE_BLOCK_MAPPING_VALUE_STATE
			return yaml_parser_process_empty_scalar(parser, event, mark)
		}
	} else if token.token_type == YAML_BLOCK_END_TOKEN {
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]
		parser.marks = parser.marks[:len(parser.marks)-1]
		*event = yaml_event_t{
			event_type: YAML_MAPPING_END_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.end_mark,
		}
		skip_token(parser)
		return true
	} else {
		mark := parser.marks[len(parser.marks)-1]
		parser.marks = parser.marks[:len(parser.marks)-1]

		return yaml_parser_set_parser_error_context(parser,
			"while parsing a block mapping", mark,
			"did not find expected key", token.start_mark)
	}
}

/*
 * Parse the productions:
 * block_mapping        ::= BLOCK-MAPPING_START
 *
 *                          ((KEY block_node_or_indentless_sequence?)?
 *
 *                          (VALUE block_node_or_indentless_sequence?)?)*
 *                           ***** *
 *                          BLOCK-END
 *
 */

func yaml_parser_parse_block_mapping_value(parser *yaml_parser_t,
	event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}

	if token.token_type == YAML_VALUE_TOKEN {
		mark := token.end_mark
		skip_token(parser)
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.token_type != YAML_KEY_TOKEN &&
			token.token_type != YAML_VALUE_TOKEN &&
			token.token_type != YAML_BLOCK_END_TOKEN {
			parser.states = append(parser.states, YAML_PARSE_BLOCK_MAPPING_KEY_STATE)
			return yaml_parser_parse_node(parser, event, true, true)
		} else {
			parser.state = YAML_PARSE_BLOCK_MAPPING_KEY_STATE
			return yaml_parser_process_empty_scalar(parser, event, mark)
		}
	} else {
		parser.state = YAML_PARSE_BLOCK_MAPPING_KEY_STATE
		return yaml_parser_process_empty_scalar(parser, event, token.start_mark)
	}
}

/*
 * Parse the productions:
 * flow_sequence        ::= FLOW-SEQUENCE-START
 *                          *******************
 *                          (flow_sequence_entry FLOW-ENTRY)*
 *                           *                   **********
 *                          flow_sequence_entry?
 *                          *
 *                          FLOW-SEQUENCE-END
 *                          *****************
 * flow_sequence_entry  ::= flow_node | KEY flow_node? (VALUE flow_node?)?
 *                          *
 */

func yaml_parser_parse_flow_sequence_entry(parser *yaml_parser_t,
	event *yaml_event_t, first bool) bool {
	if first {
		token := peek_token(parser)
		parser.marks = append(parser.marks, token.start_mark)
		skip_token(parser)
	}

	token := peek_token(parser)
	if token == nil {
		return false
	}

	if token.token_type != YAML_FLOW_SEQUENCE_END_TOKEN {
		if !first {
			if token.token_type == YAML_FLOW_ENTRY_TOKEN {
				skip_token(parser)
				token = peek_token(parser)
				if token == nil {
					return false
				}
			} else {
				mark := parser.marks[len(parser.marks)-1]
				parser.marks = parser.marks[:len(parser.marks)-1]

				return yaml_parser_set_parser_error_context(parser,
					"while parsing a flow sequence", mark,
					"did not find expected ',' or ']'", token.start_mark)
			}
		}

		if token.token_type == YAML_KEY_TOKEN {
			parser.state = YAML_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_KEY_STATE
			*event = yaml_event_t{
				event_type: YAML_MAPPING_START_EVENT,
				start_mark: token.start_mark,
				end_mark:   token.end_mark,
				implicit:   true,
				style:      yaml_style_t(YAML_FLOW_MAPPING_STYLE),
			}

			skip_token(parser)
			return true
		} else if token.token_type != YAML_FLOW_SEQUENCE_END_TOKEN {
			parser.states = append(parser.states, YAML_PARSE_FLOW_SEQUENCE_ENTRY_STATE)
			return yaml_parser_parse_node(parser, event, false, false)
		}
	}

	parser.state = parser.states[len(parser.states)-1]
	parser.states = parser.states[:len(parser.states)-1]
	parser.marks = parser.marks[:len(parser.marks)-1]

	*event = yaml_event_t{
		event_type: YAML_SEQUENCE_END_EVENT,
		start_mark: token.start_mark,
		end_mark:   token.end_mark,
	}

	skip_token(parser)
	return true
}

/*
 * Parse the productions:
 * flow_sequence_entry  ::= flow_node | KEY flow_node? (VALUE flow_node?)?
 *                                      *** *
 */

func yaml_parser_parse_flow_sequence_entry_mapping_key(parser *yaml_parser_t,
	event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}

	if token.token_type != YAML_VALUE_TOKEN &&
		token.token_type != YAML_FLOW_ENTRY_TOKEN &&
		token.token_type != YAML_FLOW_SEQUENCE_END_TOKEN {
		parser.states = append(parser.states, YAML_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_VALUE_STATE)
		return yaml_parser_parse_node(parser, event, false, false)
	} else {
		mark := token.end_mark
		skip_token(parser)
		parser.state = YAML_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_VALUE_STATE
		return yaml_parser_process_empty_scalar(parser, event, mark)
	}
}

/*
 * Parse the productions:
 * flow_sequence_entry  ::= flow_node | KEY flow_node? (VALUE flow_node?)?
 *                                                      ***** *
 */

func yaml_parser_parse_flow_sequence_entry_mapping_value(parser *yaml_parser_t,
	event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}

	if token.token_type == YAML_VALUE_TOKEN {
		skip_token(parser)
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.token_type != YAML_FLOW_ENTRY_TOKEN &&
			token.token_type != YAML_FLOW_SEQUENCE_END_TOKEN {
			parser.states = append(parser.states, YAML_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_END_STATE)
			return yaml_parser_parse_node(parser, event, false, false)
		}
	}
	parser.state = YAML_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_END_STATE
	return yaml_parser_process_empty_scalar(parser, event, token.start_mark)
}

/*
 * Parse the productions:
 * flow_sequence_entry  ::= flow_node | KEY flow_node? (VALUE flow_node?)?
 *                                                                      *
 */

func yaml_parser_parse_flow_sequence_entry_mapping_end(parser *yaml_parser_t,
	event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}

	parser.state = YAML_PARSE_FLOW_SEQUENCE_ENTRY_STATE
	*event = yaml_event_t{
		event_type: YAML_MAPPING_END_EVENT,
		start_mark: token.start_mark,
		end_mark:   token.start_mark,
	}

	return true
}

/*
 * Parse the productions:
 * flow_mapping         ::= FLOW-MAPPING-START
 *                          ******************
 *                          (flow_mapping_entry FLOW-ENTRY)*
 *                           *                  **********
 *                          flow_mapping_entry?
 *                          ******************
 *                          FLOW-MAPPING-END
 *                          ****************
 * flow_mapping_entry   ::= flow_node | KEY flow_node? (VALUE flow_node?)?
 *                          *           *** *
 */

func yaml_parser_parse_flow_mapping_key(parser *yaml_parser_t,
	event *yaml_event_t, first bool) bool {
	if first {
		token := peek_token(parser)
		parser.marks = append(parser.marks, token.start_mark)
		skip_token(parser)
	}

	token := peek_token(parser)
	if token == nil {
		return false
	}

	if token.token_type != YAML_FLOW_MAPPING_END_TOKEN {
		if !first {
			if token.token_type == YAML_FLOW_ENTRY_TOKEN {
				skip_token(parser)
				token = peek_token(parser)
				if token == nil {
					return false
				}
			} else {
				mark := parser.marks[len(parser.marks)-1]
				parser.marks = parser.marks[:len(parser.marks)-1]

				return yaml_parser_set_parser_error_context(parser,
					"while parsing a flow mapping", mark,
					"did not find expected ',' or '}'", token.start_mark)
			}
		}

		if token.token_type == YAML_KEY_TOKEN {
			skip_token(parser)
			token = peek_token(parser)
			if token == nil {
				return false
			}
			if token.token_type != YAML_VALUE_TOKEN &&
				token.token_type != YAML_FLOW_ENTRY_TOKEN &&
				token.token_type != YAML_FLOW_MAPPING_END_TOKEN {
				parser.states = append(parser.states, YAML_PARSE_FLOW_MAPPING_VALUE_STATE)
				return yaml_parser_parse_node(parser, event, false, false)
			} else {
				parser.state = YAML_PARSE_FLOW_MAPPING_VALUE_STATE
				return yaml_parser_process_empty_scalar(parser, event,
					token.start_mark)
			}
		} else if token.token_type != YAML_FLOW_MAPPING_END_TOKEN {
			parser.states = append(parser.states, YAML_PARSE_FLOW_MAPPING_EMPTY_VALUE_STATE)
			return yaml_parser_parse_node(parser, event, false, false)
		}
	}

	parser.state = parser.states[len(parser.states)-1]
	parser.states = parser.states[:len(parser.states)-1]
	parser.marks = parser.marks[:len(parser.marks)-1]
	*event = yaml_event_t{
		event_type: YAML_MAPPING_END_EVENT,
		start_mark: token.start_mark,
		end_mark:   token.end_mark,
	}

	skip_token(parser)
	return true
}

/*
 * Parse the productions:
 * flow_mapping_entry   ::= flow_node | KEY flow_node? (VALUE flow_node?)?
 *                                   *                  ***** *
 */

func yaml_parser_parse_flow_mapping_value(parser *yaml_parser_t,
	event *yaml_event_t, empty bool) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}

	if empty {
		parser.state = YAML_PARSE_FLOW_MAPPING_KEY_STATE
		return yaml_parser_process_empty_scalar(parser, event,
			token.start_mark)
	}

	if token.token_type == YAML_VALUE_TOKEN {
		skip_token(parser)
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.token_type != YAML_FLOW_ENTRY_TOKEN &&
			token.token_type != YAML_FLOW_MAPPING_END_TOKEN {
			parser.states = append(parser.states, YAML_PARSE_FLOW_MAPPING_KEY_STATE)
			return yaml_parser_parse_node(parser, event, false, false)
		}
	}

	parser.state = YAML_PARSE_FLOW_MAPPING_KEY_STATE
	return yaml_parser_process_empty_scalar(parser, event, token.start_mark)
}

/*
 * Generate an empty scalar event.
 */

func yaml_parser_process_empty_scalar(parser *yaml_parser_t, event *yaml_event_t,
	mark yaml_mark_t) bool {
	*event = yaml_event_t{
		event_type: YAML_SCALAR_EVENT,
		start_mark: mark,
		end_mark:   mark,
		value:      nil,
		implicit:   true,
		style:      yaml_style_t(YAML_PLAIN_SCALAR_STYLE),
	}

	return true
}

/*
 * Parse directives.
 */

func yaml_parser_process_directives(parser *yaml_parser_t,
	version_directive_ref **yaml_version_directive_t,
	tag_directives_ref *[]yaml_tag_directive_t) bool {

	token := peek_token(parser)
	if token == nil {
		return false
	}

	var version_directive *yaml_version_directive_t
	var tag_directives []yaml_tag_directive_t

	for token.token_type == YAML_VERSION_DIRECTIVE_TOKEN ||
		token.token_type == YAML_TAG_DIRECTIVE_TOKEN {
		if token.token_type == YAML_VERSION_DIRECTIVE_TOKEN {
			if version_directive != nil {
				yaml_parser_set_parser_error(parser,
					"found duplicate %YAML directive", token.start_mark)
				return false
			}
			if token.major != 1 ||
				token.minor != 1 {
				yaml_parser_set_parser_error(parser,
					"found incompatible YAML document", token.start_mark)
				return false
			}
			version_directive = &yaml_version_directive_t{
				major: token.major,
				minor: token.minor,
			}
		} else if token.token_type == YAML_TAG_DIRECTIVE_TOKEN {
			value := yaml_tag_directive_t{
				handle: token.value,
				prefix: token.prefix,
			}

			if !yaml_parser_append_tag_directive(parser, value, false,
				token.start_mark) {
				return false
			}
			tag_directives = append(tag_directives, value)
		}

		skip_token(parser)
		token := peek_token(parser)
		if token == nil {
			return false
		}
	}

	for i := range default_tag_directives {
		if !yaml_parser_append_tag_directive(parser, default_tag_directives[i], true, token.start_mark) {
			return false
		}
	}

	if version_directive_ref != nil {
		*version_directive_ref = version_directive
	}
	if tag_directives_ref != nil {
		*tag_directives_ref = tag_directives
	}

	return true
}

/*
 * Append a tag directive to the directives stack.
 */

func yaml_parser_append_tag_directive(parser *yaml_parser_t,
	value yaml_tag_directive_t, allow_duplicates bool, mark yaml_mark_t) bool {
	for i := range parser.tag_directives {
		tag := &parser.tag_directives[i]
		if bytes.Equal(value.handle, tag.handle) {
			if allow_duplicates {
				return true
			}
			return yaml_parser_set_parser_error(parser, "found duplicate %TAG directive", mark)
		}
	}

	parser.tag_directives = append(parser.tag_directives, value)
	return true
}