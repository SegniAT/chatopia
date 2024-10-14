// Code generated by templ - DO NOT EDIT.

// templ: version: v0.2.747
package templates

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import templruntime "github.com/a-h/templ/runtime"

func Home(title string, interests []string) templ.Component {
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
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<script>\n\t\tlet showError=(error)=>{\n\t\t\tconst errInterests = document.querySelector(\"#interests_error\")\n\t\t\tconst errInterestsClassList = errInterests.classList\n\t\t\terrInterests.innerText = error\n\t\t\terrInterestsClassList.remove(\"hidden\")\n\t\t\terrInterestsClassList.add(\"block\")\n\t\t}\n\n\t\tlet hideError=()=> {\n\t\t\tconst errInterests = document.querySelector(\"#interests_error\")\n\t\t\tconst errInterestsClassList = errInterests.classList\n\t\t\terrInterests.innerText = \"\"\n\t\t\terrInterestsClassList.add(\"hidden\")\n\t\t\terrInterestsClassList.remove(\"block\")\n\t\t}\n\n\t\tlet validateInterest = (interest) => {\n\t\t\tconst interest_pills = document.querySelector(\"#interest_pills\")\n\t\t\tconst errInterestsClassList = document.querySelector(\"#interests_error\").classList\n\n\t\t\t// check if 3<\n\t\t\tif (interest_pills.childElementCount == 3) {\n\t\t\t\tthrow \"maximum of only 3 interests allowed\"\n\t\t\t}else {\n\t\t\t\thideError()\n\t\t\t}\n\t\t\tlet newInterest = interest.trim().toLowerCase()\n\t\t\tif(newInterest.length == 0) {\n\t\t\t\tthrow \"interest too short (min 1 character)\"\n\t\t\t}\n\n\t\t\tif (newInterest.length > 25) {\n\t\t\t\tthrow \"interest too long (max 25 characters)\"\n\t\t\t}\n\n\t\t\treturn newInterest\n\t\t}\n\n\t\tlet addInterestPill = (newInterest) => {\n\t\t\tconst interestPill = document.createElement(\"div\")\n\t\t\tconst interestPillRemove = document.createElement(\"span\")\n\t\t\tinterestPillRemove .classList.add(\"remove_interest_pill\")\n\t\t\tinterestPillRemove.textContent = \"x\"\n\t\n\t\t\tlet pillStyle = \"flex justify-center items-center pl-2 h-6 bg-chatopia-2 border-l border-chatopia-3 shadow-sm shadow-chatopia-3 rounded-md\"\n\t\t\tlet pillRemoveStyle = \"flex items-center font-bold text-md h-full ml-2 px-1 rounded-md rounded-l-none bg-red-600 text-chatopia-5 cursor-pointer\"\t\n\n\t\t\tinterestPill.classList.add(...pillStyle.split(\" \"))\n\t\t\tinterestPillRemove.classList.add(...pillRemoveStyle.split(\" \"))\n\n\t\t\tinterestPill.textContent = newInterest\n\t\t\tinterestPill.appendChild(interestPillRemove)\n\n\t\t\tinterest_pills.appendChild(interestPill)\n\t\t\t\n\t\t\tinterestPillRemove.addEventListener(\"click\", () => {\n\t\t\t\tinterestPill.remove()\n\t\t\t})\n\t\t}\n\n\t\tlet initializeInterests = () => {\n\t\t\tconst interest_pills = document.querySelector(\"#interest_pills\")\n\t\t\tconst interestsInput = document.querySelector(\"#interests_input\")\n\n\t\t\tif (interestsInput){\n\t\t\t\tinterestsInput.addEventListener(\"keydown\", (e) => {\n\t\t\t\t\tif (e.key===\"Enter\"){\n\t\t\t\t\t\te.preventDefault()\n\n\t\t\t\t\t\tlet newInterest = e.target.value\n\n\t\t\t\t\t\ttry {\n\t\t\t\t\t\t\tnewInterest = validateInterest(newInterest)\n\t\t\t\t\t\t}catch(err){\n\t\t\t\t\t\t\tshowError(err)\n\t\t\t\t\t\t\treturn\n\t\t\t\t\t\t}\n\n\t\t\t\t\t\taddInterestPill(newInterest)\n\t\t\t\t\t\te.target.value = \"\"\n\t\t\t\t\t}\n\t\t\t\t});\n\t\t\t}\n\n\t\t\tconst remove_interest_pill = document.querySelectorAll(\".remove_interest_pill\")\n\n\t\t\tremove_interest_pill.forEach(removeSpan => {\n\t\t\t\tremoveSpan.addEventListener(\"click\", (e) => {\n\t\t\t\t\tconst pill = e.target.parentElement\n\t\t\t\t\tif (pill) {\n\t\t\t\t\t\tpill.remove()\n\t\t\t\t\t}\n\t\t\t\t})\n\t\t\t})\n\t\t}\n\n\t\tlet getInterestsFromPills=()=>{\n\t\t\tconst interest_pills = document.querySelector(\"#interest_pills\")\n\t\t\tlet interests = []\n\n\t\t\tif(!interest_pills) {\n\t\t\t\tconsole.log(\"can't find interest pills container element\")\n\t\t\t\treturn interests\n\t\t\t}\n\n\t\t\tinterest_pills.childNodes.forEach(node => {\n\t\t\t\tlet text = node.textContent\n\t\t\t\tif(text.length > 0) {\n\t\t\t\t\t// cut out the 'X'\n\t\t\t\t\ttext = text.slice(0, text.length-1)\n\t\t\t\t}\n\t\t\t\tinterests.push(text)\n\t\t\t})\n\t\t\treturn interests\n\t\t}\n\n\t\tlet updateInterestsLocalStorage=()=>{\n\t\t\tlet interests = getInterestsFromPills()\n\t\t\tlocalStorage.setItem(\"interests\", JSON.stringify(interests))\n\t\t}\n\n\t\tlet loadInterestsFromLocalStorage=()=>{\n\t\t\tlet savedInterests = localStorage.getItem(\"interests\")\n\t\t\tif (!savedInterests) {\n\t\t\t\treturn\n\t\t\t}\n\n\t\t\tsavedInterests = JSON.parse(savedInterests)\n\t\t\tlet len = savedInterests.length\n\n\t\t\tif(len> 3) {\n\t\t\t\tsavedInterests = savedInterests.slice(0, 3)\n\t\t\t\treturn\n\t\t\t}\n\n\t\t\tsavedInterests.forEach(interest => addInterestPill(interest))\n\t\t}\n\n\t\tdocument.addEventListener(\"DOMContentLoaded\",(e)=> {\n\t\t\tconst interestsInput = document.querySelector(\"#interests_input\")\n\t\t\tconst remove_interest_pill = document.querySelector(\".remove_interest_pill\")\n\n\t\t\t// load previously submitted interests\n\t\t\tloadInterestsFromLocalStorage(interestsInput)\n\t\t\t\n\t\t\t// Reinitialize the interests listeners on first load\n\t\t\tinitializeInterests()\n\n\t\t\t// Reinitialize the interests listeners when the form is re-rendered\n\t\t\tdocument.querySelector(\"#interest_form\").addEventListener(\"htmx:afterSwap\", (e) => {\n\t\t\t\tinitializeInterests()\n\t\t\t})\n\n\t\t\t// remember the users interests\n\t\t\tdocument.querySelectorAll(\"#video_chat_form, #text_chat_form\").forEach(form => {\n\t\t\t\tform.addEventListener(\"htmx:beforeRequest\", (e) => {\n\t\t\t\t\tupdateInterestsLocalStorage()\n\t\t\t\t});\n\n\t\t\t})\n\t\t});\n\n\t</script>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Var2 := templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
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
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<div class=\"pb-10 pt-20 flex flex-col lg:flex-row justify-between items-center\"><div class=\"lg:order-2\"><img class=\"max-w-lg\" src=\"/assets/logo.png\" alt=\"chatopia logo\"></div><div class=\"lg:order-1\"><h1 class=\"text-4xl lg:text-5xl py-20 font-thin\">Looking for love (or just a good laugh)? Get on <span class=\"text-chatopia-1 font-bold\">Chatopia</span>, Ethiopia's answer to boredom! </h1></div></div><div><p class=\"\">Start chatting:</p><div class=\"flex justify-start gap-2 p-2\"><form id=\"video_chat_form\" hx-post=\"/chat/video\" hx-target=\"#interest_form\" hx-vals=\"js:{\n\t\t\t\t\t\t&#34;interests[]&#34;: getInterestsFromPills()\n\t\t\t\t\t}\"><button class=\"rounded-md border-2 border-chatopia-1 bg-chatopia-2 text-chatopia-1 py-1 px-4 font-semibold\" type=\"submit\">Video</button></form><form id=\"text_chat_form\" hx-post=\"/chat/text\" hx-target=\"#interest_form\" hx-vals=\"js:{\n\t\t\t\t\t\t&#34;interests[]&#34;: getInterestsFromPills()\n\t\t\t\t\t}\"><button class=\"rounded-md border-2 border-chatopia-3 bg-chatopia-3 text-chatopia-5 py-1 px-5 font-semibold\" type=\"submit\">Text</button></form></div><div id=\"interest_form\">")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = InterestInput(interests, nil).Render(ctx, templ_7745c5c3_Buffer)
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</div></div>")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			return templ_7745c5c3_Err
		})
		templ_7745c5c3_Err = Base(title).Render(templ.WithChildren(ctx, templ_7745c5c3_Var2), templ_7745c5c3_Buffer)
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return templ_7745c5c3_Err
	})
}

func InterestInput(interests []string, interestsError error) templ.Component {
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
		templ_7745c5c3_Var3 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var3 == nil {
			templ_7745c5c3_Var3 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<p>Interests (optional):</p><div class=\"flex flex-col items-start\"><div id=\"interest_pills\" class=\"flex gap-2 py-2\">")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		for _, interest := range interests {
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<span class=\"flex justify-center items-center pl-2 h-6 bg-chatopia-2 border-l border-chatopia-3 shadow-sm shadow-chatopia-3 rounded-md\">")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var4 string
			templ_7745c5c3_Var4, templ_7745c5c3_Err = templ.JoinStringErrs(interest)
			if templ_7745c5c3_Err != nil {
				return templ.Error{Err: templ_7745c5c3_Err, FileName: `ui/templates/home.templ`, Line: 214, Col: 15}
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var4))
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(" <span class=\"flex items-center font-bold text-md h-full ml-2 px-1 rounded-md rounded-l-none bg-red-600 text-chatopia-5 cursor-pointer\">x</span></span>")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</div><textarea id=\"interests_input\" class=\"border border-chatopia-3 focus:border-chatopia-1 p-2 bg-chatopia-2 resize\" placeholder=\"Enter your interests...\" maxlength=\"25\"></textarea> <span id=\"interests_error\" class=\"block text-red-600 text-sm\">")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		if interestsError != nil {
			var templ_7745c5c3_Var5 string
			templ_7745c5c3_Var5, templ_7745c5c3_Err = templ.JoinStringErrs(interestsError.Error())
			if templ_7745c5c3_Err != nil {
				return templ.Error{Err: templ_7745c5c3_Err, FileName: `ui/templates/home.templ`, Line: 222, Col: 28}
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var5))
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</span></div>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return templ_7745c5c3_Err
	})
}
