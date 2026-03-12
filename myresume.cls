% The MIT License (MIT)
%
% Copyright (c) 2021 Zackary Crosley <https://crosleyzack.com>
%
% Permission is hereby granted, free of charge, to any person obtaining a copy
% of this software and associated documentation files (the "Software"), to deal
% in the Software without restriction, including without limitation the rights
% to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
% copies of the Software, and to permit persons to whom the Software is
% furnished to do so, subject to the following conditions:
%
% The above copyright notice and this permission notice shall be included in
% all copies or substantial portions of the Software.
%
% THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
% IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
% FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
% AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
% LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
% OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
% THE SOFTWARE.

\NeedsTeXFormat{LaTeX2e}
\ProvidesClass{myresume}[2021-10-23 Resume template for Zack Crosley]
% Base this off the article class
\LoadClass{article}
% Set font family.
\RequirePackage{lmodern}
\RequirePackage[T1]{fontenc}
\renewcommand{\familydefault}{\sfdefault}
% set page margins.
\RequirePackage[margin=0.75in]{geometry}
% titlesec to format sections, titles.
\RequirePackage[explicit]{titlesec}
% Add tcolorbox for colorful sections, titles
\RequirePackage{xcolor}
% Multi column package for condensed sections
\RequirePackage{multicol}
\RequirePackage{enumitem}
% Set line spacing
\RequirePackage{setspace}
% hyperlinks
\RequirePackage[hidelinks]{hyperref}
% Font awesome for social logos.
\RequirePackage{fontawesome6}

%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%
% Constants for use in template.
% Edit here to adjust template!
\definecolor{accent}{RGB}{209, 99, 73}
\definecolor{secondary}{RGB}{45, 45, 45}
\newcommand{\resumeLineSpacing}{1.05}
%   Font sizes
\newcommand{\mainFontSize}{12pt}
\newcommand{\sectionFontSize}{12pt}
\newcommand{\subsectionFontSize}{12pt}
\newcommand{\subtitleFontSize}{12pt}
\newcommand{\titleFontSize}{26pt}
%   Itemize object
\newcommand{\itemizeItemSeparation}{1.1mm}
\newcommand{\itemizeSubItemSeparation}{0.75mm}
\newcommand{\beforeItemizeSpacing}{1mm}
\newcommand{\afterItemizeSpacing}{1mm}
\newcommand{\beforeSubItemizeSpacing}{0.5mm}
\newcommand{\afterSubItemizeSpacing}{0.5mm}
\newcommand{\bulletIndent}{1mm}
%   Multicol object
\newcommand{\multicolSpacing}{1mm}
%   Delimiter spacing
\newcommand{\subtitleDelimSpacing}{3mm}
\newcommand{\paragraphDelimSpacing}{0mm}
%   Section object
\newcommand{\sectionColor}{white}
\newcommand{\sectionBgPadding}{0.75mm}
\newcommand{\beforeSection}{0mm}
\newcommand{\afterSection}{0mm}
%   SubSection Object
\newcommand{\beforeSubSection}{0mm}
\newcommand{\afterSubSection}{1mm}
%   Paragraph Object
\newcommand{\beforeParagraph}{0mm}
\newcommand{\afterParagraph}{0mm}

%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%

%   Remove Other spacing
%   NOTE even with all spacing set to 0, there is still inherent
%        spacing from using \titleformat with display style.
%        Unfortunately, simply switching to block or any other style
%        creates different problems.
\setlength{\parindent}{0mm}
\setlength{\parskip}{0mm}
\setlength{\baselineskip}{0mm}
\setlength{\abovedisplayskip}{0mm}
\setlength{\belowdisplayskip}{0mm}
\setlength{\abovedisplayshortskip}{0mm}
\setlength{\belowdisplayshortskip}{0mm}
\setlength{\topskip}{0mm}
\setlength{\multicolsep}{\multicolSpacing}
\lineskiplimit=-\maxdimen

% Set article line spacing [Value specified above]
\setstretch{\resumeLineSpacing}

% set spacing for itemized lists
\setlist[itemize]{leftmargin=*,
                  nosep}
\setlist[itemize, 1]{before=\vspace{\beforeItemizeSpacing},
                     after=\vspace{\afterItemizeSpacing},
                     labelsep=\bulletIndent,
                     label=\textbullet,
                     itemsep=\itemizeItemSeparation}
\setlist[itemize, 2]{before=\vspace{\beforeSubItemizeSpacing},
                     after=\vspace{\afterSubItemizeSpacing},
                     labelsep=\bulletIndent,
                     label=\textbullet,
                     itemsep=\itemizeSubItemSeparation}


% Declare counter for use in template
\newcounter{mycount}

% Set resume default color.
\color{secondary}

% Format section header
\newcommand{\sectionfont}{\fontsize{\sectionFontSize}{\sectionFontSize}\color{\sectionColor}\bfseries}
\setlength{\fboxsep}{\sectionBgPadding}
\setlength{\fboxrule}{0mm}
\titlespacing*{\section}{0mm}{\beforeSection}{\afterSection}
\titleformat{\section}
  [display]
  {\sectionfont}
  {}
  {0mm}
  {\colorbox{accent}{\fbox{\parbox{\textwidth}{\MakeUppercase{#1}}}}}

% Format subsection header
\newcommand{\subsectionfont}{\fontsize{\subsectionFontSize}{\subsectionFontSize}\color{accent}}
\titlespacing*{\subsection}{0mm}{\beforeSubSection}{\afterSubSection}
\titleformat{\subsection}
  [display]
  {\subsectionfont}
  {}
  {0mm}
  {\MakeUppercase{#1}}

% Format paragraph header
\newcommand{\paragraphfont}{\fontsize{\mainFontSize}{\mainFontSize}\color{secondary}}
\titlespacing*{\paragraph}{0mm}{\beforeParagraph}{\afterParagraph}
%   looper to print out items.
\newcommand{\printheader}[1]{\setcounter{mycount}{0}
                             \@for\tmp:=#1\do{
                                \ifnum \value{mycount}>0 {\hspace{\paragraphDelimSpacing}·\hspace{\paragraphDelimSpacing}} \fi
                                \stepcounter{mycount}
                                \tmp}}
\titleformat{\paragraph}
  [display]
  {\paragraphfont}
  {}
  {0mm}
  {\printheader{#1}}

% Format Title
\newcommand{\titlefont}{\fontsize{\titleFontSize}{\titleFontSize}\color{accent}\normalfont\MakeUppercase}
\newcommand{\resumename}[1]{\begin{flushleft}
                              \titlefont{#1}
                            \end{flushleft}}

% Format Subtitle
\newcommand{\subtitlefont}{\fontsize{\subtitleFontSize}{\subtitleFontSize}\color{secondary}\normalfont}
%     looper to print out subtitle.
\newcommand{\printsubtitle}[1]{\setcounter{mycount}{0}
                               \@for\stmp:=#1\do{
                                  \ifnum \value{mycount}>0 {\hspace{\subtitleDelimSpacing}|\hspace{\subtitleDelimSpacing}} \fi
                                  \stepcounter{mycount}
                                  \MakeUppercase{\stmp}}}
\newcommand{\resumecontact}[1]{\subtitlefont{\printsubtitle{#1}}}

% Print out a list of items in multiple columns.
\newcommand{\skills}[1]{\begin{multicols}{5}
                            \begin{itemize}
                              \@for\ltmp:=#1\do{
                                \item[] \ltmp}
                            \end{itemize}
                        \end{multicols}}
