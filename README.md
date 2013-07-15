**Background**
---------------

[æ³¨é³ç¬¦è](http://en.wikipedia.org/wiki/Zhuyin ""), ZhÃ¹yÄ«n fÃºhÃ o, ãã¨Ë ã§ã£ ãã¨Ë ãã Ë, or bopomofo

Is a phonetic "alphabet" that assists in the pronunciation of Chinese characters. It is nearly identical in purpose to [Pinyin](http://en.wikipedia.org/wiki/Pinyin "Pinyin") in that it assists pronunciation by phonetically "spelling out" Chinese characters. The only difference is that it uses 37 unique symbols, rather than the 26 symbols of the English alphabet.

While haven fallen out of favor with the Mainland, it is still widely used as an input method in Taiwan and other places.

It is my opinion that it should be widely re-adopted. With 37 unique characters, it captures sounds that the 26 characters of the English alphabet and combinations thereof cannot. By doing so, it avoids an issue with Pinyin, where those learning Chinese with the English alphabet, fall victim to using pre-conditioned pronunciation of English letters- inevitably resulting in an accent. 



**The Problem**
---------------
The layout of Zhuyin on the modern keyboard seemed have been added as an after-thought. As seen below, the alphabet is placed in sequential order, from top to bottom, left to right, with no regards to frequency of occurrence or any other considerations. There is no apparent rhyme or reason for this layout, and it certainly does not assist in typing speeds or adoption. 


![Zhuyin Keyboard](http://upload.wikimedia.org/wikipedia/commons/f/fa/Keyboard_layout_Zhuyin.svg "Zhuyin Keyboard")



Granted, this situation is mitigated by the very order of the alphabet, where all the consonants are located at the beginning, with vowels
at the end. When laid out in sequential order on the keyboard, it conveniently concentrates the consonants on the left, with the vowels on the right. However, there is no denying that speed bonuses can be garnered by having frequently appearing consonants and connective vowels placed under the default assumed position of fingers on a QWERTY keyboard: "ASDF JKL:"



**The Solution**
---------------

I hereby propose improving this input method by injecting a bit of common sense in terms of key placement.

### Design Considerations
1. Effort will be made minimize changes- to prevent burdening users from having to learn an entirely new keyboard format altogether.
2. Number keys 1,2,3,4,5 are to be vacated of any characters to allow logical input of tonal numbers.
3. Consideration will also be given to the use of extended pinyin characters to allow the input of Chinese characters by other dialects.

### Components
1. **The Collection** - In this initial stage, the text from a large body of modern and classic Chinese literature will be retrieved and collected- with equal weight given to all Chinese-speaking regions and with a 70/30 split between modern vernacular and [Literary Chinese](http://en.wikipedia.org/wiki/Classical_Chinese "Literary Chinese").
2. **[The Converter](https://blog.robertjchen.net/post-5/pinyin--%3E-zhuyin-conversion "")** - Next, with all the text on hand, a simple script will be written to look up all the Chinese characters in the [Unicode Unihan Database](http://www.unicode.org/charts/unihan.html "Unicode Unihan Database"). As the database has Hanyu Pinyin data for all the characters, but none for Zhuyin, the Pinyin will be extracted and converted into Zhuyin.
3. **The Frequency Analysis** - With all the characters now represented in Zhuyin, a script will simply bin and draw a histogram of all Zhuyin characters. This information will roughly reveal the frequency of occurrence of each Zhuyin character in the Chinese language.
4. **The Keyboard** - Armed with the frequency information, a new keyboard layout will be devised such that those symbols that are used more often, are placed under the default finger placement on a keyboard. Obviously, there will be much leeway as to where certain characters are placed (say, if they had the same frequency of occurrence)... these will have to be left up to discretionary arbitration.
5. **The Frontend** - Next, a web-based IME will be created as a demonstration. This will simply consist of a JS library. When activated, this IME will perform like any other Chinese IME. It will also include a soft keyboard for reference as to character placement, similar to the image above.
6. **The Backend** - The Go-based service will handle WebSocket requests from The Frontend. In addition to returning the actual character represented by the keyed-in Zhuyin, it will also build character frequency profiles for better autocomplete, serve a light dictionary, and much more. Such data will be pulled from a pre-constructed sqlite database using the data garnered from the previous steps.

Everthing- All the components, material sources, notes, rationalizations, will be open source for examination, criticism, and academic review. I am open to all suggestions and input, as it is crucial that all things are considered when establishing a standard.

The project repository will be found here: [https://github.com/fernjager/rational-ime](https://github.com/fernjager/rational-ime "https://github.com/fernjager/rational-ime")
