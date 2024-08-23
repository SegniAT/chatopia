// Code generated by templ - DO NOT EDIT.

// templ: version: v0.2.747
package templates

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import templruntime "github.com/a-h/templ/runtime"

func TextChat() templ.Component {
	return templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
		templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
		if !templ_7745c5c3_IsBuffer {
			defer func() {
				templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err == nil {
					templ_7745c5c3_Err = templ_7745c5c3_BufErr
				}
			}()
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var1 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var1 == nil {
			templ_7745c5c3_Var1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<script>\n\t\tdocument.addEventListener(\"DOMContentLoaded\",(e)=> {\n\t\t\tlet chatForm = document.querySelector(\"#chat_form\")\n\t\t\t\n\t\t\tif(chatForm) {\n\t\t\t\tchatForm.addEventListener('htmx:wsAfterSend', function(e) {\n\t\t\t\t\tlet chatMessage = chatForm.querySelector(\"#chat_message\")\n\t\t\t\t\ttry {\n\t\t\t\t\t\tlet messageData = JSON.parse(e.detail.message)\n\t\t\t\t\t\tif(messageData.message_type != \"typing\" && chatMessage){\n\t\t\t\t\t\t\tchatMessage.value = \"\"\n\t\t\t\t\t\t}\n\t\t\t\t\t} catch(err) {\n\t\t\t\t\t\tconsole.error(\"Error parsing ws message JSON:\", err)\n\t\t\t\t\t}\n\t\t\t\t\t\n\t\t\t\t});\n\t\t\t}\n\t\t\t\n\t\t})\n\t</script><div><div id=\"connection_status\"><p>Connecting...</p></div><div class=\"flex-grow p-2 my-2 shadow-sm shadow-chatopia-1 rounded-md min-h-20\"><div id=\"chat_bubbles\" class=\"flex flex-col gap-1 pb-1\"></div><div id=\"chat_typing\"></div></div><div><form id=\"chat_form\" class=\"flex items-end gap-2\" ws-send hx-trigger=\"submit\" hx-vals=\"{&#34;message_type&#34;:&#34;chat_message&#34;}\"><textarea id=\"chat_message\" name=\"chat_message\" class=\"rounded-md text-chatopia-2 p-2 resize\" maxlength=\"150\" ws-send hx-vals=\"{&#34;message_type&#34;:&#34;typing&#34;}\" hx-trigger=\"keyup changed throttle:3s\"></textarea> <button id=\"send_chat_button\" type=\"submit\" class=\"px-2 bg-chatopia-1 text-chatopia-2 rounded-md\">Send</button></form></div></div>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return templ_7745c5c3_Err
	})
}

func ChatBubble(message string, self bool) templ.Component {
	return templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
		templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
		if !templ_7745c5c3_IsBuffer {
			defer func() {
				templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err == nil {
					templ_7745c5c3_Err = templ_7745c5c3_BufErr
				}
			}()
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var2 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var2 == nil {
			templ_7745c5c3_Var2 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<div id=\"chat_bubbles\" hx-swap-oob=\"beforeend\"><div class=\"flex gap-1\">")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		if self {
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<p class=\"text-blue-500\">You:</p>")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
		} else {
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<p class=\"text-red-500\">Stranger:</p>")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<p>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var3 string
		templ_7745c5c3_Var3, templ_7745c5c3_Err = templ.JoinStringErrs(message)
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `ui/templates/text_chat.templ`, Line: 65, Col: 13}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var3))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</p></div></div>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return templ_7745c5c3_Err
	})
}
