<!DOCTYPE html>
<html lang="en">
  <head>
    <!--page:idefix-->
    <meta http-equiv="X-UA-Compatible" content="IE=Edge" />
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
    <meta http-equiv="Pragma" content="no-cache" />
    <meta http-equiv="Cache-Control" content="no-cache" />
    <meta http-equiv="Expires" content="-1" />
    <link rel="shortcut icon" href="images/favicon.png" />
    <link rel="icon" href="images/favicon.png" />
    <title>X-RAY Server</title>
    <link rel="stylesheet" type="text/css" href="index_style.css" />
    <link rel="stylesheet" type="text/css" href="form_style.css" />
    <link rel="stylesheet" type="text/css" href="/js/table/table.css" />

    <script
      language="JavaScript"
      type="text/javascript"
      src="/js/jquery.js"
    ></script>
    <script
      language="JavaScript"
      type="text/javascript"
      src="/js/httpApi.js"
    ></script>
    <script
      language="JavaScript"
      type="text/javascript"
      src="/state.js"
    ></script>
    <script
      language="JavaScript"
      type="text/javascript"
      src="/general.js"
    ></script>
    <script
      language="JavaScript"
      type="text/javascript"
      src="/popup.js"
    ></script>
    <script
      language="JavaScript"
      type="text/javascript"
      src="/help.js"
    ></script>
    <script
      language="JavaScript"
      type="text/javascript"
      src="/validator.js"
    ></script>
    <script type="module" crossorigin src="/user/idefix/app.js"></script>
  </head>

  <body>
    <div id="TopBanner"></div>
    <div id="Loading" class="popup_bg"></div>
    <table class="content" align="center" cellpadding="0" cellspacing="0">
      <tbody>
        <tr>
          <td width="17">&nbsp;</td>
          <td valign="top" width="202">
            <div id="mainMenu"></div>
            <div id="subMenu"></div>
          </td>
          <td valign="top">
            <div id="tabMenu" class="submenuBlock"></div>
            <table
              width="98%"
              border="0"
              align="left"
              cellpadding="0"
              cellspacing="0"
            >
              <tbody>
                <tr>
                  <td valign="top">
                    <table
                      width="760px"
                      border="0"
                      cellpadding="4"
                      cellspacing="0"
                      id="FormTitle"
                      class="FormTitle"
                    >
                      <tbody>
                        <tr bgcolor="#4D595D">
                          <td valign="top">
                            <div class="formfontdesc">
                              <div>&nbsp;</div>
                              <div id="root"></div>
                            </div>
                          </td>
                        </tr>
                      </tbody>
                    </table>
                  </td>
                </tr>
              </tbody>
            </table>
          </td>
        </tr>
      </tbody>
    </table>
    <div id="footer"></div>

    <script>
      document.addEventListener("DOMContentLoaded", function () {
        if (typeof window.show_menu === "function") {
          window.show_menu();
        }
      });
    </script>
  </body>
</html>
