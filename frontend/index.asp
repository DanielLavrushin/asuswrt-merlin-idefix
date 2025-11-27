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

    <script language="JavaScript" type="text/javascript" src="/js/jquery.js"></script>
    <script language="JavaScript" type="text/javascript" src="/js/httpApi.js"></script>
    <script language="JavaScript" type="text/javascript" src="/state.js"></script>
    <script language="JavaScript" type="text/javascript" src="/general.js"></script>
    <script language="JavaScript" type="text/javascript" src="/popup.js"></script>
    <script language="JavaScript" type="text/javascript" src="/help.js"></script>
    <script language="JavaScript" type="text/javascript" src="/validator.js"></script>
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
            <div id="root"></div>
          </td>
        </tr>
      </tbody>
    </table>
    <div id="footer"></div>

    <script>
      const custom_settings_raw = '<% get_custom_settings(); %>';
      const stripAnsi = /\u001b\[[0-9;]*m/g;
      const stripControls = /[\u0000-\u001F]+/g;
      const custom_settings = JSON.parse(custom_settings_raw.replace(stripAnsi, '').replace(stripControls, ''));

      window.idefix = {
        custom_settings
      };
      window.showLoading = function () {};
      window.hideLoading = function () {};
      document.addEventListener('DOMContentLoaded', function () {
        if (typeof window.show_menu === 'function') {
          window.show_menu();
        }
      });
    </script>
    <script>
      document.write('<script type="module" crossorigin src="/user/idefix/app.js?v=' + Date.now() + '"><\/script>');
    </script>
  </body>
</html>
